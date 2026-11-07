// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveSeriesGroundTruth checks extraction against real products. It is
// skipped unless CAMEL_LIVE_ASINS is set, for example:
//
//	CAMEL_LIVE_ASINS=us:B07T32R43P,au:B00OYGGMXK go test -run Live ./internal/camel/
func TestLiveSeriesGroundTruth(t *testing.T) {
	list := os.Getenv("CAMEL_LIVE_ASINS")
	if list == "" {
		t.Skip("set CAMEL_LIVE_ASINS to run live checks")
	}
	c := NewClient(30 * time.Second)
	ctx := context.Background()
	checked, matched := 0, 0
	for _, item := range strings.Split(list, ",") {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("bad item %q", item)
		}
		p, err := c.FetchProduct(ctx, Target{ASIN: parts[1], Locale: parts[0]})
		if err != nil {
			t.Errorf("%s: %v", item, err)
			continue
		}
		all, err := c.Series(ctx, p, PriceTypes, time.Now())
		if err != nil {
			t.Errorf("%s: series: %v", item, err)
			continue
		}
		for _, ps := range p.Prices {
			if !ps.HasData() || ps.Lowest == nil || ps.Highest == nil {
				continue
			}
			segs := all[ps.Type]
			okLo := segmentNear(segs, ps.LowestDate, *ps.Lowest, 0.01, 7)
			okHi := segmentNear(segs, ps.HighestDate, *ps.Highest, 0.01, 7)
			t.Logf("%s %s: %d segments; lowest %.2f@%s found=%v; highest %.2f@%s found=%v", item, ps.Type, len(segs), *ps.Lowest, ps.LowestDate, okLo, *ps.Highest, ps.HighestDate, okHi)
			checked += 2
			if okLo {
				matched++
			}
			if okHi {
				matched++
			}
		}
	}
	// Overlapping lines can hide one extreme; require 90% of extremes.
	if checked > 0 && float64(matched)/float64(checked) < 0.9 {
		t.Errorf("matched %d of %d lifetime extremes", matched, checked)
	}
	t.Logf("matched %d of %d lifetime extremes", matched, checked)
}
