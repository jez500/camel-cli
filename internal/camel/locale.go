// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

// Package camel holds the camelcamelcamel domain logic: Amazon URL parsing,
// marketplace (locale) mapping, product-page parsing, RSS feed parsing, the
// deal verdict, and price-series extraction from chart images.
package camel

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Locale describes one Amazon marketplace that camelcamelcamel tracks.
type Locale struct {
	Code         string // short code used by the CLI and the chart host: us, au, ...
	CamelHost    string // camelcamelcamel host for this marketplace
	AmazonDomain string // amazon domain without "www."
	Currency     string // ISO currency code
}

var locales = map[string]Locale{
	"us": {Code: "us", CamelHost: "camelcamelcamel.com", AmazonDomain: "amazon.com", Currency: "USD"},
	"au": {Code: "au", CamelHost: "au.camelcamelcamel.com", AmazonDomain: "amazon.com.au", Currency: "AUD"},
	"uk": {Code: "uk", CamelHost: "uk.camelcamelcamel.com", AmazonDomain: "amazon.co.uk", Currency: "GBP"},
	"ca": {Code: "ca", CamelHost: "ca.camelcamelcamel.com", AmazonDomain: "amazon.ca", Currency: "CAD"},
	"de": {Code: "de", CamelHost: "de.camelcamelcamel.com", AmazonDomain: "amazon.de", Currency: "EUR"},
	"fr": {Code: "fr", CamelHost: "fr.camelcamelcamel.com", AmazonDomain: "amazon.fr", Currency: "EUR"},
	"it": {Code: "it", CamelHost: "it.camelcamelcamel.com", AmazonDomain: "amazon.it", Currency: "EUR"},
	"es": {Code: "es", CamelHost: "es.camelcamelcamel.com", AmazonDomain: "amazon.es", Currency: "EUR"},
}

// localeAliases maps common spellings to a locale code.
var localeAliases = map[string]string{
	"usa": "us", "com": "us", "amazon.com": "us",
	"aus": "au", "australia": "au", "amazon.com.au": "au",
	"gb": "uk", "amazon.co.uk": "uk",
	"amazon.ca": "ca", "amazon.de": "de", "amazon.fr": "fr", "amazon.it": "it", "amazon.es": "es",
}

// LookupLocale returns the locale for a code or alias (case-insensitive).
func LookupLocale(code string) (Locale, error) {
	c := strings.ToLower(strings.TrimSpace(code))
	if a, ok := localeAliases[c]; ok {
		c = a
	}
	if l, ok := locales[c]; ok {
		return l, nil
	}
	return Locale{}, fmt.Errorf("unknown locale %q (supported: %s)", code, strings.Join(LocaleCodes(), ", "))
}

// LocaleCodes returns the supported locale codes in a stable order.
func LocaleCodes() []string {
	out := make([]string, 0, len(locales))
	for k := range locales {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// localeForAmazonHost maps an Amazon host (any subdomain) to a locale.
func localeForAmazonHost(host string) (Locale, bool) {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	h = strings.TrimPrefix(h, "www.")
	h = strings.TrimPrefix(h, "smile.")
	h = strings.TrimPrefix(h, "m.")
	// Longest domain first so amazon.com.au wins over amazon.com.
	best := Locale{}
	for _, l := range locales {
		if h == l.AmazonDomain || strings.HasSuffix(h, "."+l.AmazonDomain) {
			if len(l.AmazonDomain) > len(best.AmazonDomain) {
				best = l
			}
		}
	}
	return best, best.Code != ""
}

// localeForCamelHost maps a camelcamelcamel host to a locale.
func localeForCamelHost(host string) (Locale, bool) {
	h := strings.ToLower(strings.TrimPrefix(host, "www."))
	for _, l := range locales {
		if h == l.CamelHost {
			return l, true
		}
	}
	return Locale{}, false
}

// ProductURL returns the camelcamelcamel product page URL.
func (l Locale) ProductURL(asin string) string {
	return "https://" + l.CamelHost + "/product/" + asin
}

// AmazonURL returns the Amazon product URL.
func (l Locale) AmazonURL(asin string) string {
	u := url.URL{Scheme: "https", Host: "www." + l.AmazonDomain, Path: "/dp/" + asin}
	return u.String()
}
