// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Target is a resolved product reference: one ASIN on one marketplace.
type Target struct {
	ASIN   string `json:"asin"`
	Locale string `json:"locale"`
	// ShortLink is set when the input was a short link (a.co, amzn.to) that
	// must be resolved over HTTP before the ASIN is known.
	ShortLink string `json:"short_link,omitempty"`
}

// ErrNoASIN means the input contained no recognizable ASIN.
var ErrNoASIN = errors.New("no ASIN found")

var (
	asinRe      = regexp.MustCompile(`^[A-Za-z0-9]{10}$`)
	asinInPath  = regexp.MustCompile(`(?i)/(?:dp|gp/product|gp/aw/d|gp/offer-listing|exec/obidos/asin|exec/obidos/tg/detail/-|o|product|d)/([A-Z0-9]{10})(?:[/?#]|$)`)
	shortHosts  = map[string]bool{"a.co": true, "amzn.to": true, "amzn.eu": true, "amzn.asia": true, "amzn.com": true}
	shortAmzCom = regexp.MustCompile(`(?i)^/([A-Z0-9]{10})/?$`)
)

// IsASIN reports whether s looks like an ASIN (10 alphanumerics).
func IsASIN(s string) bool { return asinRe.MatchString(strings.TrimSpace(s)) }

// ParseInput turns an ASIN, Amazon URL, camelcamelcamel URL, or short link
// into a Target. defaultLocale is used for bare ASINs and short links.
// It makes no network calls; short links come back with ShortLink set.
func ParseInput(input, defaultLocale string) (Target, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return Target{}, fmt.Errorf("%w: empty input", ErrNoASIN)
	}
	def, err := LookupLocale(defaultLocale)
	if err != nil {
		return Target{}, err
	}
	if IsASIN(s) {
		return Target{ASIN: strings.ToUpper(s), Locale: def.Code}, nil
	}
	if !strings.Contains(s, "://") {
		// Accept "amazon.com/dp/..." without a scheme.
		if strings.Contains(s, ".") && strings.Contains(s, "/") {
			s = "https://" + s
		} else {
			return Target{}, fmt.Errorf("%w in %q: pass a 10-character ASIN or an Amazon product URL", ErrNoASIN, input)
		}
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return Target{}, fmt.Errorf("%w in %q: not a valid URL", ErrNoASIN, input)
	}
	host := strings.ToLower(u.Hostname())
	hostNoWWW := strings.TrimPrefix(host, "www.")

	if shortHosts[hostNoWWW] {
		// amzn.com/<ASIN> is an old direct form; the rest need a redirect.
		if hostNoWWW == "amzn.com" {
			if m := shortAmzCom.FindStringSubmatch(u.Path); m != nil {
				return Target{ASIN: strings.ToUpper(m[1]), Locale: "us"}, nil
			}
		}
		u.Scheme = "https" // never resolve a short link over plain http
		return Target{Locale: def.Code, ShortLink: u.String()}, nil
	}

	var loc Locale
	if l, ok := localeForAmazonHost(host); ok {
		loc = l
	} else if l, ok := localeForCamelHost(host); ok {
		loc = l
	} else if strings.Contains(host, "amazon.") {
		return Target{}, fmt.Errorf("unsupported Amazon marketplace %q (supported: %s)", host, strings.Join(LocaleCodes(), ", "))
	} else {
		return Target{}, fmt.Errorf("%w: %q is not an Amazon or camelcamelcamel URL", ErrNoASIN, host)
	}

	if m := asinInPath.FindStringSubmatch(u.Path); m != nil {
		return Target{ASIN: strings.ToUpper(m[1]), Locale: loc.Code}, nil
	}
	for _, key := range []string{"asin", "ASIN", "pd_rd_i"} {
		if v := u.Query().Get(key); IsASIN(v) {
			return Target{ASIN: strings.ToUpper(v), Locale: loc.Code}, nil
		}
	}
	return Target{}, fmt.Errorf("%w in %q: the URL has no /dp/<ASIN> or /gp/product/<ASIN> part", ErrNoASIN, input)
}

// ParseRedirectLocation extracts a Target from the Location header of a
// short-link redirect.
func ParseRedirectLocation(location, defaultLocale string) (Target, error) {
	t, err := ParseInput(location, defaultLocale)
	if err != nil {
		return Target{}, fmt.Errorf("%w: short link did not lead to an Amazon product (%v)", ErrNoASIN, err)
	}
	if t.ShortLink != "" {
		return Target{}, fmt.Errorf("%w: short link redirected to another short link %q", ErrNoASIN, location)
	}
	return t, nil
}

// AllowedRedirectHost reports whether a short-link redirect may go to host:
// another short-link host, an Amazon marketplace, or a camelcamelcamel host.
func AllowedRedirectHost(host string) bool {
	h := strings.ToLower(strings.TrimPrefix(host, "www."))
	if shortHosts[h] {
		return true
	}
	if _, ok := localeForAmazonHost(host); ok {
		return true
	}
	_, ok := localeForCamelHost(host)
	return ok
}
