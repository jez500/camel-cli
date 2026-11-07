// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// Drop is one item from the top price drops feed.
type Drop struct {
	ASIN       string   `json:"asin"`
	Title      string   `json:"title"`
	DropPct    *float64 `json:"drop_pct"`
	DropAmount *float64 `json:"drop_amount"`
	Price      *float64 `json:"price"`
	OldPrice   *float64 `json:"old_price"`
	URL        string   `json:"url"`
	PubDate    string   `json:"published,omitempty"`
	GUID       string   `json:"guid,omitempty"`
}

// PopularItem is one item from the popular products feed.
type PopularItem struct {
	ASIN      string   `json:"asin"`
	Title     string   `json:"title"`
	Current   *float64 `json:"current"`
	ListPrice *float64 `json:"list_price"`
	Average   *float64 `json:"average"`
	BestPrice bool     `json:"best_price"`
	// BelowAvgPct is how far the current price is below the average (positive = cheaper).
	BelowAvgPct *float64 `json:"below_avg_pct"`
	URL         string   `json:"url"`
	PubDate     string   `json:"published,omitempty"`
}

type rss struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			GUID        string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

var (
	reDropTitle = regexp.MustCompile(`^(.*?)\s+-\s+down\s+([0-9.,]+)%\s+\(([^)]*)\)\s+to\s+(.+?)\s+from\s+(.+?)\s*$`)
	reLinkASIN  = regexp.MustCompile(`/product/([A-Z0-9]{10})`)
	reCurrent   = regexp.MustCompile(`(?i)Current Price:\s*([^<-]+?)\s*(-\s*Best Price)?\s*<`)
	reList      = regexp.MustCompile(`(?i)List Price:\s*([^<]+?)\s*<`)
	reAvg       = regexp.MustCompile(`(?i)Avg\.? Price:\s*([^<]+?)\s*<`)
)

func ptr(v float64) *float64 { return &v }

func priceOrNil(s string) *float64 {
	if v, ok := ParsePrice(s); ok {
		return &v
	}
	return nil
}

func feedDate(s string) string {
	s = strings.TrimSpace(s)
	for _, l := range []string{time.RFC1123Z, time.RFC1123, "Mon, 02 Jan 2006 15:04:05 MST", "Mon, 2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return s
}

// ParseDropsFeed parses the /top_drops/feed RSS document.
func ParseDropsFeed(body []byte) ([]Drop, error) {
	var doc rss
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing drops feed: %w", err)
	}
	out := make([]Drop, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		d := Drop{Title: strings.TrimSpace(StripControl(html.UnescapeString(it.Title))), URL: strings.TrimSpace(it.Link), PubDate: feedDate(it.PubDate), GUID: strings.TrimSpace(it.GUID)}
		if m := reLinkASIN.FindStringSubmatch(it.Link); m != nil {
			d.ASIN = m[1]
		}
		if m := reDropTitle.FindStringSubmatch(d.Title); m != nil {
			d.Title = m[1]
			if v, ok := ParsePrice(m[2]); ok {
				d.DropPct = ptr(v)
			}
			d.DropAmount = priceOrNil(m[3])
			d.Price = priceOrNil(m[4])
			d.OldPrice = priceOrNil(m[5])
		}
		out = append(out, d)
	}
	return out, nil
}

// ParsePopularFeed parses the /popular.xml RSS document.
func ParsePopularFeed(body []byte) ([]PopularItem, error) {
	var doc rss
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing popular feed: %w", err)
	}
	out := make([]PopularItem, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		p := PopularItem{Title: strings.TrimSpace(StripControl(html.UnescapeString(it.Title))), URL: strings.TrimSpace(it.Link), PubDate: feedDate(it.PubDate)}
		if m := reLinkASIN.FindStringSubmatch(it.Link); m != nil {
			p.ASIN = m[1]
		}
		desc := html.UnescapeString(it.Description) + "<"
		if m := reCurrent.FindStringSubmatch(desc); m != nil {
			p.Current = priceOrNil(m[1])
			p.BestPrice = m[2] != ""
		}
		if m := reList.FindStringSubmatch(desc); m != nil {
			p.ListPrice = priceOrNil(m[1])
		}
		if m := reAvg.FindStringSubmatch(desc); m != nil {
			p.Average = priceOrNil(m[1])
		}
		if p.Current != nil && p.Average != nil && *p.Average > 0 {
			v := round2((*p.Average - *p.Current) / *p.Average * 100)
			p.BelowAvgPct = &v
		}
		out = append(out, p)
	}
	return out, nil
}

func round2(v float64) float64 {
	if v < 0 {
		return -float64(int64(-v*100+0.5)) / 100
	}
	return float64(int64(v*100+0.5)) / 100
}
