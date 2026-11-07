// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"errors"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Price types tracked by camelcamelcamel.
const (
	TypeAmazon = "amazon"
	TypeNew    = "new"
	TypeUsed   = "used"
)

// PriceTypes lists the price types in display order.
var PriceTypes = []string{TypeAmazon, TypeNew, TypeUsed}

// PriceTypeLabel returns the site's label for a price type.
func PriceTypeLabel(t string) string {
	switch t {
	case TypeAmazon:
		return "Amazon"
	case TypeNew:
		return "3rd Party New"
	case TypeUsed:
		return "3rd Party Used"
	}
	return t
}

// ErrNotTracked means camelcamelcamel has no page for the ASIN on that
// marketplace (the site redirects untracked products to its home page).
var ErrNotTracked = errors.New("not tracked by camelcamelcamel")

// PriceStats is one row of the camelcamelcamel summary table.
type PriceStats struct {
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Lowest      *float64 `json:"lowest"`
	LowestDate  string   `json:"lowest_date,omitempty"`
	Highest     *float64 `json:"highest"`
	HighestDate string   `json:"highest_date,omitempty"`
	Current     *float64 `json:"current"`
	CurrentDate string   `json:"current_date,omitempty"`
	Average     *float64 `json:"average"`
	// CurrentText keeps the raw cell when the current price is not a number
	// (for example "Not in Stock").
	CurrentText string `json:"current_text,omitempty"`
}

// HasData reports whether the row has any price.
func (p PriceStats) HasData() bool {
	return p.Lowest != nil || p.Highest != nil || p.Current != nil || p.Average != nil
}

// Product is the parsed camelcamelcamel product page.
type Product struct {
	ASIN          string            `json:"asin"`
	Locale        string            `json:"locale"`
	Title         string            `json:"title"`
	URL           string            `json:"url"`
	AmazonURL     string            `json:"amazon_url"`
	ImageURL      string            `json:"image_url,omitempty"`
	ProductGroup  string            `json:"product_group,omitempty"`
	Category      string            `json:"category,omitempty"`
	Manufacturer  string            `json:"manufacturer,omitempty"`
	Model         string            `json:"model,omitempty"`
	UPC           string            `json:"upc,omitempty"`
	EAN           string            `json:"ean,omitempty"`
	ListPrice     *float64          `json:"list_price,omitempty"`
	LastChecked   string            `json:"last_checked,omitempty"`
	TrackingSince string            `json:"tracking_since,omitempty"`
	RangeStarts   map[string]string `json:"range_starts,omitempty"`
	Prices        []PriceStats      `json:"prices"`
}

// Price returns the stats row for a price type.
func (p *Product) Price(t string) (PriceStats, bool) {
	for _, s := range p.Prices {
		if s.Type == t {
			return s, true
		}
	}
	return PriceStats{}, false
}

var (
	reOGTitle   = regexp.MustCompile(`<meta property="og:title" content="([^"]*)"`)
	reOGImage   = regexp.MustCompile(`<meta property="og:image" content="([^"]*)"`)
	reCanonical = regexp.MustCompile(`<link rel="canonical" href="([^"]*)"`)
	reKeywords  = regexp.MustCompile(`<meta name="keywords" content="([^"]*)"`)
	reLegend    = regexp.MustCompile(`(?s)<div class="table-scroll camelegend">(.*?)</table>`)
	reRow       = regexp.MustCompile(`(?s)<tr class="pt ([a-z]+)[^"]*" data-field="([a-z]+)">(.*?)</tr>`)
	reCell      = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	reSmall     = regexp.MustCompile(`(?s)<small>\s*\(?([^<)]*)\)?\s*</small>`)
	reLastCheck = regexp.MustCompile(`title="[^"]*?(?:Last checked|checked)\s+([^".]+?)\.?"`)
	reRangeTag  = regexp.MustCompile(`<a[^>]*id="toggle_tp_summary_(1m|3m|6m|1y|all)"[^>]*>`)
	reRangeFrom = regexp.MustCompile(`going back to ([^"]+?)"`)
	reDetail    = regexp.MustCompile(`(?s)<strong>([^<]+)</strong>\s*</td>\s*<td[^>]*>(.*?)</td>`)
	reTags      = regexp.MustCompile(`<[^>]+>`)
	reSpace     = regexp.MustCompile(`\s+`)
)

func cleanHTMLText(s string) string {
	s = reTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ")
	return strings.TrimSpace(reSpace.ReplaceAllString(StripControl(s), " "))
}

// StripControl removes control characters (for example ESC from a decoded
// "&#27;") so scraped text cannot inject terminal escape sequences.
func StripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// ParseProductPage parses a camelcamelcamel product page. locale and asin
// come from the request; the page does not always repeat them reliably.
func ParseProductPage(body []byte, asin, locale string) (*Product, error) {
	s := string(body)
	legend := reLegend.FindStringSubmatch(s)
	if legend == nil {
		return nil, ErrNotTracked
	}
	loc, err := LookupLocale(locale)
	if err != nil {
		return nil, err
	}
	p := &Product{
		ASIN:        asin,
		Locale:      loc.Code,
		URL:         loc.ProductURL(asin),
		AmazonURL:   loc.AmazonURL(asin),
		RangeStarts: map[string]string{},
	}
	if m := reOGTitle.FindStringSubmatch(s); m != nil {
		p.Title = cleanHTMLText(m[1])
	}
	if m := reOGImage.FindStringSubmatch(s); m != nil {
		p.ImageURL = html.UnescapeString(m[1])
	}
	if m := reCanonical.FindStringSubmatch(s); m != nil && strings.Contains(m[1], "/product/") {
		p.URL = html.UnescapeString(m[1])
	}

	// Detail table (labels in English on all sites tested; meta keywords as fallback).
	for _, m := range reDetail.FindAllStringSubmatch(s, -1) {
		key := strings.ToLower(cleanHTMLText(m[1]))
		val := cleanHTMLText(m[2])
		switch key {
		case "product group":
			p.ProductGroup = val
		case "category":
			p.Category = val
		case "manufacturer":
			p.Manufacturer = val
		case "model":
			p.Model = val
		case "upc":
			p.UPC = val
		case "ean":
			p.EAN = val
		case "list price":
			if v, ok := ParsePrice(val); ok {
				p.ListPrice = &v
			}
		}
	}
	if m := reKeywords.FindStringSubmatch(s); m != nil {
		kw := html.UnescapeString(m[1])
		fill := func(dst *string, key string) {
			if *dst != "" {
				return
			}
			re := regexp.MustCompile(regexp.QuoteMeta(key) + `: ([^,]+)`)
			if mm := re.FindStringSubmatch(kw); mm != nil {
				*dst = strings.TrimSpace(mm[1])
			}
		}
		fill(&p.UPC, "UPC")
		fill(&p.EAN, "EAN")
		fill(&p.Model, "Model")
		fill(&p.ProductGroup, "Product group")
		fill(&p.Category, "Category")
	}

	if m := reLastCheck.FindStringSubmatch(legend[1]); m != nil {
		p.LastChecked = NormalizeDate(m[1])
	}
	for _, m := range reRangeTag.FindAllStringSubmatch(s, -1) {
		if f := reRangeFrom.FindStringSubmatch(m[0]); f != nil {
			p.RangeStarts[m[1]] = NormalizeDate(f[1])
		}
	}
	p.TrackingSince = p.RangeStarts["all"]

	for _, row := range reRow.FindAllStringSubmatch(legend[1], -1) {
		t := row[2]
		cells := reCell.FindAllStringSubmatch(row[3], -1)
		if len(cells) < 5 {
			continue
		}
		ps := PriceStats{Type: t, Label: PriceTypeLabel(t)}
		ps.Lowest, ps.LowestDate, _ = parsePriceCell(cells[1][1])
		ps.Highest, ps.HighestDate, _ = parsePriceCell(cells[2][1])
		var curText string
		ps.Current, ps.CurrentDate, curText = parsePriceCell(cells[3][1])
		if ps.Current == nil && curText != "" && curText != "-" {
			ps.CurrentText = curText
		}
		ps.Average, _, _ = parsePriceCell(cells[4][1])
		p.Prices = append(p.Prices, ps)
	}
	if len(p.Prices) == 0 {
		return nil, ErrNotTracked
	}
	return p, nil
}

// parsePriceCell returns the price, the ISO date in <small>, and the raw text.
func parsePriceCell(cell string) (*float64, string, string) {
	date := ""
	if m := reSmall.FindStringSubmatch(cell); m != nil {
		date = NormalizeDate(m[1])
		cell = reSmall.ReplaceAllString(cell, "")
	}
	text := cleanHTMLText(cell)
	if v, ok := ParsePrice(text); ok {
		return &v, date, text
	}
	return nil, date, text
}

var rePriceNum = regexp.MustCompile(`[0-9][0-9.,\s\x{00a0}]*`)

// ParsePrice parses prices such as "$1,234.56", "A$19.98", "£9.99",
// "19,99 €", "1.234,56 €". It returns false for "-" or text without digits.
func ParsePrice(s string) (float64, bool) {
	m := rePriceNum.FindString(s)
	if m == "" {
		return 0, false
	}
	n := strings.NewReplacer(" ", "", " ", "").Replace(strings.TrimSpace(m))
	n = strings.TrimRight(n, ".,")
	lastDot := strings.LastIndex(n, ".")
	lastComma := strings.LastIndex(n, ",")
	switch {
	case lastComma > lastDot:
		// Comma is the decimal separator when 1-2 digits follow it.
		if len(n)-lastComma-1 <= 2 {
			n = strings.ReplaceAll(n[:lastComma], ".", "") + "." + n[lastComma+1:]
			n = strings.ReplaceAll(n, ",", "")
		} else {
			n = strings.ReplaceAll(n, ",", "")
		}
	case lastComma < 0 && lastDot >= 0 && len(n)-lastDot-1 == 3:
		// "1.234" with no comma: dots are thousands separators (de/fr/it/es).
		n = strings.ReplaceAll(n, ".", "")
	default:
		n = strings.ReplaceAll(n, ",", "")
	}
	v, err := strconv.ParseFloat(n, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

var dateLayouts = []string{"Jan 02, 2006", "Jan 2, 2006", "January 2, 2006", "2 Jan 2006", "02 Jan 2006", "2006-01-02", "02.01.2006", "02/01/2006"}

// NormalizeDate converts site dates ("Jun 26, 2026") to ISO 8601. Unknown
// formats are returned trimmed and unchanged.
func NormalizeDate(s string) string {
	s = strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "()."))
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

// ParseISODate parses a date produced by NormalizeDate.
func ParseISODate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}
