// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jez500/camel-cli/internal/cliutil"
)

// UserAgent is sent on every request. camelcamelcamel's Cloudflare rules let
// this honest, unique agent through and challenge generic or spoofed ones,
// so do not change it to a browser string.
const UserAgent = "camel-cli/0.1.0"

const acceptHeader = "text/html,application/xhtml+xml,application/xml;q=0.9,image/png,*/*;q=0.8"

// ErrChallenged means Cloudflare answered with a bot challenge instead of content.
var ErrChallenged = errors.New("camelcamelcamel returned a Cloudflare challenge (HTTP 403)")

// Client fetches camelcamelcamel pages, feeds and chart images.
type Client struct {
	HTTP    *http.Client
	limiter *cliutil.AdaptiveLimiter
}

// NewClient returns a client limited to about one request per second.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		// Go's standard transport on purpose: browser TLS fingerprints get challenged.
		HTTP:    &http.Client{Timeout: timeout},
		limiter: cliutil.NewAdaptiveLimiter(1.0),
	}
}

// Response is a fetched document.
type Response struct {
	Body     []byte
	FinalURL string
	Type     string
}

// Get fetches a URL, following redirects, with one retry on HTTP 429.
func (c *Client) Get(ctx context.Context, rawURL string) (*Response, error) {
	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", acceptHeader)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", rawURL, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close() // body already fully read; a close error carries no information
		if readErr != nil {
			return nil, fmt.Errorf("reading %s: %w", rawURL, readErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			c.limiter.OnRateLimit()
			if attempt == 0 {
				continue
			}
			return nil, &cliutil.RateLimitError{URL: rawURL, RetryAfter: cliutil.RetryAfter(resp), Body: truncateBody(body)}
		}
		if resp.StatusCode == http.StatusForbidden && (resp.Header.Get("cf-mitigated") != "" || strings.Contains(string(body), "Just a moment")) {
			return nil, fmt.Errorf("%w for %s; wait a few minutes and retry", ErrChallenged, rawURL)
		}
		if resp.StatusCode >= 400 {
			return nil, &HTTPError{URL: rawURL, Status: resp.StatusCode}
		}
		c.limiter.OnSuccess()
		return &Response{Body: body, FinalURL: resp.Request.URL.String(), Type: resp.Header.Get("Content-Type")}, nil
	}
}

// HTTPError is a non-2xx response that is not a challenge or rate limit.
type HTTPError struct {
	URL    string
	Status int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("GET %s returned HTTP %d", e.URL, e.Status)
}

func truncateBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// FetchProduct fetches and parses the product page. It returns ErrNotTracked
// when the site redirects away from the product page.
func (c *Client) FetchProduct(ctx context.Context, t Target) (*Product, error) {
	loc, err := LookupLocale(t.Locale)
	if err != nil {
		return nil, err
	}
	resp, err := c.Get(ctx, loc.ProductURL(t.ASIN))
	if err != nil {
		var he *HTTPError
		if errors.As(err, &he) && he.Status == http.StatusNotFound {
			return nil, fmt.Errorf("%s on %s: %w", t.ASIN, loc.CamelHost, ErrNotTracked)
		}
		return nil, err
	}
	if !strings.Contains(resp.FinalURL, "/product/") {
		return nil, fmt.Errorf("%s on %s: %w", t.ASIN, loc.CamelHost, ErrNotTracked)
	}
	p, err := ParseProductPage(resp.Body, t.ASIN, loc.Code)
	if err != nil {
		if errors.Is(err, ErrNotTracked) {
			return nil, fmt.Errorf("%s on %s: %w", t.ASIN, loc.CamelHost, ErrNotTracked)
		}
		return nil, err
	}
	return p, nil
}

// FetchChart downloads a chart PNG.
func (c *Client) FetchChart(ctx context.Context, o ChartOptions) ([]byte, string, error) {
	u, err := ChartURL(o)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.Get(ctx, u)
	if err != nil {
		return nil, u, err
	}
	if !strings.HasPrefix(resp.Type, "image/") {
		return nil, u, fmt.Errorf("chart request returned %q, not an image", resp.Type)
	}
	return resp.Body, u, nil
}

// ResolveShortLink reads the redirect target of a short link (a.co, amzn.to)
// without loading the Amazon page itself.
func (c *Client) ResolveShortLink(ctx context.Context, t Target) (Target, error) {
	if t.ShortLink == "" {
		return t, nil
	}
	hc := &http.Client{
		Timeout: c.HTTP.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Never follow a redirect off Amazon, short-link or camelcamelcamel hosts.
			if req.URL.Scheme != "https" || !AllowedRedirectHost(req.URL.Hostname()) {
				return fmt.Errorf("short link redirected to an unexpected host %q", req.URL.Host)
			}
			if _, err := ParseRedirectLocation(req.URL.String(), t.Locale); err == nil {
				return http.ErrUseLastResponse
			}
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.ShortLink, nil)
	if err != nil {
		return Target{}, err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := hc.Do(req)
	if err != nil {
		return Target{}, fmt.Errorf("resolving short link %s: %w", t.ShortLink, err)
	}
	_ = resp.Body.Close() // only the redirect Location is needed; body is unused
	loc := resp.Header.Get("Location")
	if loc == "" {
		loc = resp.Request.URL.String()
	}
	if resp.Request != nil && resp.Request.URL != nil {
		if ref, perr := resp.Request.URL.Parse(loc); perr == nil {
			loc = ref.String()
		}
	}
	return ParseRedirectLocation(loc, t.Locale)
}
