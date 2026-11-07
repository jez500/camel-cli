// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseInput(t *testing.T) {
	cases := []struct {
		in, def, asin, locale string
		short                 bool
		wantErr               bool
	}{
		{"B07T32R43P", "us", "B07T32R43P", "us", false, false},
		{"b07t32r43p", "au", "B07T32R43P", "au", false, false},
		{"https://www.amazon.com/dp/B07T32R43P", "us", "B07T32R43P", "us", false, false},
		{"https://www.amazon.com.au/Alpine-PWE-S8-Restyle/dp/B00OYGGMXK/ref=sr_1_1?keywords=x", "us", "B00OYGGMXK", "au", false, false},
		{"https://www.amazon.co.uk/gp/product/B07T32R43P?th=1", "us", "B07T32R43P", "uk", false, false},
		{"https://www.amazon.de/gp/aw/d/B07T32R43P", "us", "B07T32R43P", "de", false, false},
		{"amazon.ca/dp/B07T32R43P", "us", "B07T32R43P", "ca", false, false},
		{"https://au.camelcamelcamel.com/product/B00OYGGMXK", "us", "B00OYGGMXK", "au", false, false},
		{"https://camelcamelcamel.com/product/B07T32R43P?active=price_amazon", "au", "B07T32R43P", "us", false, false},
		{"https://a.co/d/abc1234", "au", "", "au", true, false},
		{"https://amzn.to/3xyzabc", "us", "", "us", true, false},
		{"https://www.amazon.co.jp/dp/B07T32R43P", "us", "", "", false, true},
		{"https://example.com/dp/B07T32R43P", "us", "", "", false, true},
		{"https://www.amazon.com/s?k=kindle", "us", "", "", false, true},
		{"kindle paperwhite", "us", "", "", false, true},
		{"", "us", "", "", false, true},
	}
	for _, c := range cases {
		got, err := ParseInput(c.in, c.def)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseInput(%q) = %+v, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseInput(%q) error %v", c.in, err)
			continue
		}
		if got.ASIN != c.asin || got.Locale != c.locale || (got.ShortLink != "") != c.short {
			t.Errorf("ParseInput(%q) = %+v, want asin=%s locale=%s short=%v", c.in, got, c.asin, c.locale, c.short)
		}
	}
}

func TestParseRedirectLocation(t *testing.T) {
	got, err := ParseRedirectLocation("https://www.amazon.com.au/dp/B00OYGGMXK?ref=x", "us")
	if err != nil || got.ASIN != "B00OYGGMXK" || got.Locale != "au" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if _, err := ParseRedirectLocation("https://a.co/d/other", "us"); err == nil {
		t.Fatal("want error for redirect to another short link")
	}
}

func TestLookupLocale(t *testing.T) {
	for in, want := range map[string]string{"US": "us", "australia": "au", "amazon.co.uk": "uk", "gb": "uk"} {
		l, err := LookupLocale(in)
		if err != nil || l.Code != want {
			t.Errorf("LookupLocale(%q) = %v, %v", in, l.Code, err)
		}
	}
	if _, err := LookupLocale("jp"); err == nil {
		t.Error("want error for unsupported locale")
	}
}

func TestParsePrice(t *testing.T) {
	cases := map[string]float64{
		"$19.98": 19.98, "$1,234.56": 1234.56, "£9.99": 9.99, "19,99 €": 19.99,
		"1.234,56 €": 1234.56, "1.234 €": 1234, "$39<wbr />.95": 0, "A$330.25": 330.25, "$4,397.00": 4397,
	}
	for in, want := range cases {
		if strings.Contains(in, "<") {
			continue
		}
		got, ok := ParsePrice(in)
		if !ok || math.Abs(got-want) > 1e-9 {
			t.Errorf("ParsePrice(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"-", "", "Not in Stock"} {
		if _, ok := ParsePrice(bad); ok {
			t.Errorf("ParsePrice(%q) should fail", bad)
		}
	}
}

func TestNormalizeDate(t *testing.T) {
	if got := NormalizeDate("Jun 26, 2026"); got != "2026-06-26" {
		t.Errorf("got %q", got)
	}
	if got := NormalizeDate("(Jan 02, 2020)"); got != "2020-01-02" {
		t.Errorf("got %q", got)
	}
	if got := NormalizeDate("26. Juni 2026"); got != "26. Juni 2026" {
		t.Errorf("unknown formats must pass through, got %q", got)
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func f(p *float64) float64 {
	if p == nil {
		return math.NaN()
	}
	return *p
}

func loadUSProduct(t *testing.T) *Product {
	t.Helper()
	p, err := ParseProductPage(mustRead(t, "product_us_B07T32R43P.html"), "B07T32R43P", "us")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseProductPageUS(t *testing.T) {
	p := loadUSProduct(t)
	if p.Title != "Local Hero (The Criterion Collection) [Blu-ray]" {
		t.Errorf("title %q", p.Title)
	}
	if p.UPC != "715515234610" || p.EAN != "071551523461" {
		t.Errorf("upc/ean %q %q", p.UPC, p.EAN)
	}
	if p.ProductGroup != "DVD" || p.Category != "Blu-ray" {
		t.Errorf("group/category %q %q", p.ProductGroup, p.Category)
	}
	if p.ListPrice == nil || *p.ListPrice != 39.95 {
		t.Errorf("list price %v", p.ListPrice)
	}
	if p.LastChecked != "2026-11-06" || p.TrackingSince != "2019-06-19" {
		t.Errorf("dates %q %q", p.LastChecked, p.TrackingSince)
	}
	if len(p.RangeStarts) != 5 {
		t.Errorf("range starts %v", p.RangeStarts)
	}
	want := map[string][4]float64{
		"amazon": {19.98, 39.99, 19.98, 26.78},
		"new":    {14.00, 79.99, 19.98, 36.67},
		"used":   {11.50, 70.44, 25.52, 25.66},
	}
	if len(p.Prices) != 3 {
		t.Fatalf("prices %d", len(p.Prices))
	}
	for _, ps := range p.Prices {
		w := want[ps.Type]
		got := [4]float64{f(ps.Lowest), f(ps.Highest), f(ps.Current), f(ps.Average)}
		if got != w {
			t.Errorf("%s = %v, want %v", ps.Type, got, w)
		}
	}
	am, _ := p.Price("amazon")
	if am.LowestDate != "2026-06-26" || am.HighestDate != "2025-10-17" || am.CurrentDate != "2026-11-06" {
		t.Errorf("amazon dates %+v", am)
	}
}

func TestParseProductPageAUEmptyRow(t *testing.T) {
	p, err := ParseProductPage(mustRead(t, "product_au_B00OYGGMXK.html"), "B00OYGGMXK", "au")
	if err != nil {
		t.Fatal(err)
	}
	used, ok := p.Price("used")
	if !ok || used.HasData() {
		t.Errorf("used row should exist and be empty: %+v", used)
	}
	am, _ := p.Price("amazon")
	if f(am.Lowest) != 330.25 || am.LowestDate != "2020-01-02" {
		t.Errorf("amazon %+v", am)
	}
	if !strings.HasPrefix(p.AmazonURL, "https://www.amazon.com.au/") {
		t.Errorf("amazon url %s", p.AmazonURL)
	}
}

func TestParseProductPageNotTracked(t *testing.T) {
	if _, err := ParseProductPage([]byte("<html><title>home</title></html>"), "B000000000", "us"); err != ErrNotTracked {
		t.Fatalf("want ErrNotTracked, got %v", err)
	}
}

func daysApart(a, b string) int {
	ta, _ := ParseISODate(a)
	tb, _ := ParseISODate(b)
	d := ta.Sub(tb).Hours() / 24
	return int(math.Abs(d))
}

// segmentNear finds a segment whose span is within slack days of date and
// whose price is within tol of price.
func segmentNear(segs []Segment, date string, price, tol float64, slack int) bool {
	for _, s := range segs {
		inside := s.From <= date && date <= s.To
		if (inside || daysApart(s.From, date) <= slack || daysApart(s.To, date) <= slack) && math.Abs(s.Price-price) <= tol {
			return true
		}
	}
	return false
}

func TestExtractSeriesMatchesPageFacts(t *testing.T) {
	p := loadUSProduct(t)
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	cal, err := CalibrationFor(p, now)
	if err != nil {
		t.Fatal(err)
	}
	all, err := ExtractSeries(mustRead(t, "chart_us_B07T32R43P_all.png"), cal, PriceTypes)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range PriceTypes {
		ps, _ := p.Price(typ)
		segs := all[typ]
		if len(segs) < 5 {
			t.Fatalf("%s: only %d segments", typ, len(segs))
		}
		// Ground truth from the page: lowest and highest prices near their dates.
		if !segmentNear(segs, ps.LowestDate, *ps.Lowest, 0.01, 7) {
			t.Errorf("%s: no segment at lowest %.2f near %s", typ, *ps.Lowest, ps.LowestDate)
		}
		if !segmentNear(segs, ps.HighestDate, *ps.Highest, 0.01, 7) {
			t.Errorf("%s: no segment at highest %.2f near %s", typ, *ps.Highest, ps.HighestDate)
		}
		last := segs[len(segs)-1]
		if last.To != "2026-11-06" || last.Price != *ps.Current {
			t.Errorf("%s: last segment %+v, want current %.2f on 2026-11-06", typ, last, *ps.Current)
		}
		for _, s := range segs {
			if s.Price < *ps.Lowest-0.001 || s.Price > *ps.Highest+0.001 {
				t.Errorf("%s: segment %+v outside lifetime range", typ, s)
			}
			if s.From > s.To {
				t.Errorf("%s: segment %+v inverted", typ, s)
			}
		}
	}
	if daysApart(all["amazon"][0].From, "2019-06-19") > 3 {
		t.Errorf("amazon starts %s, want about 2019-06-19", all["amazon"][0].From)
	}
}

func TestExtractSeriesRejectsNonChart(t *testing.T) {
	cal := Calibration{Types: map[string]TypeCal{"amazon": {Highest: 2, Lowest: 1}}, Start: time.Now().AddDate(-1, 0, 0), End: time.Now()}
	if _, err := ExtractSeries([]byte("not a png"), cal, PriceTypes); err == nil {
		t.Fatal("want decode error")
	}
}

func TestDailyAndFilter(t *testing.T) {
	segs := []Segment{
		{From: "2026-01-01", To: "2026-01-02", Price: 10, Available: true},
		{From: "2026-01-05", To: "2026-01-06", Price: 8, Available: false},
	}
	d := Daily(segs)
	if len(d) != 6 || d[2].Date != "2026-01-03" || d[2].Price != 10 || d[4].Price != 8 || d[4].Available {
		t.Fatalf("daily = %+v", d)
	}
	fs := FilterSegments(segs, "2026-01-02")
	if len(fs) != 2 || fs[0].From != "2026-01-02" {
		t.Fatalf("filter = %+v", fs)
	}
	if len(Daily(nil)) != 0 {
		t.Fatal("empty daily must be empty slice")
	}
}

func TestChartURL(t *testing.T) {
	u, err := ChartURL(ChartOptions{Locale: "au", ASIN: "B00OYGGMXK", Types: []string{"new,amazon"}, Range: "1y"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "https://charts.camelcamelcamel.com/au/B00OYGGMXK/amazon-new.png?") || !strings.Contains(u, "tp=1y") {
		t.Errorf("url %s", u)
	}
	if _, err := ChartURL(ChartOptions{Locale: "us", ASIN: "B00OYGGMXK", Range: "2y"}); err == nil {
		t.Error("want range error")
	}
	if _, err := NormalizeTypes([]string{"refurb"}); err == nil {
		t.Error("want type error")
	}
}

func TestParseDropsFeed(t *testing.T) {
	drops, err := ParseDropsFeed(mustRead(t, "drops_feed.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(drops) == 0 {
		t.Fatal("no drops")
	}
	d := drops[0]
	if d.ASIN == "" || d.DropPct == nil || d.Price == nil || d.OldPrice == nil || d.DropAmount == nil {
		t.Fatalf("incomplete drop %+v", d)
	}
	if math.Abs(*d.OldPrice-*d.Price-*d.DropAmount) > 0.02 {
		t.Errorf("amount mismatch %+v", d)
	}
	if strings.Contains(d.Title, " - down ") {
		t.Errorf("title not stripped: %q", d.Title)
	}
}

func TestParsePopularFeed(t *testing.T) {
	items, err := ParsePopularFeed(mustRead(t, "popular_feed.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("no items")
	}
	first := items[0]
	if first.ASIN != "B00JAQ5NC0" || f(first.Current) != 36.99 || f(first.ListPrice) != 55.99 || f(first.Average) != 46.59 || !first.BestPrice {
		t.Errorf("first = %+v", first)
	}
	if first.BelowAvgPct == nil || math.Abs(*first.BelowAvgPct-20.6) > 0.1 {
		t.Errorf("below avg %v", first.BelowAvgPct)
	}
}

func TestAssess(t *testing.T) {
	v := func(x float64) *float64 { return &x }
	cases := []struct {
		ps   PriceStats
		want string
	}{
		{PriceStats{Current: v(19.98), Lowest: v(19.98), Average: v(26.78), Highest: v(39.99)}, VerdictBuy},
		{PriceStats{Current: v(20.5), Lowest: v(19.98), Average: v(26.78)}, VerdictBuy},
		{PriceStats{Current: v(25), Lowest: v(19.98), Average: v(26.78)}, VerdictGood},
		{PriceStats{Current: v(28), Lowest: v(19.98), Average: v(26.78)}, VerdictFair},
		{PriceStats{Current: v(25.52), Lowest: v(11.50), Average: v(25.66)}, VerdictFair},
		{PriceStats{Current: v(35), Lowest: v(19.98), Average: v(26.78)}, VerdictWait},
		{PriceStats{CurrentText: "Not in Stock"}, VerdictUnknown},
	}
	for _, c := range cases {
		if got := Assess(c.ps).Verdict; got != c.want {
			t.Errorf("Assess(%v) = %s, want %s", f(c.ps.Current), got, c.want)
		}
	}
	best, ok := Best([]Deal{Assess(cases[2].ps), Assess(cases[0].ps), Assess(cases[5].ps)})
	if !ok || f(best.Current) != 19.98 {
		t.Errorf("best = %+v", best)
	}
}
