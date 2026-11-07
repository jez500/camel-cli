// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import "fmt"

// Verdict values.
const (
	VerdictBuy     = "buy"     // at or within 3% of the lowest price ever
	VerdictGood    = "good"    // at least 5% below the average price
	VerdictFair    = "fair"    // from 5% below to 10% above the average
	VerdictWait    = "wait"    // more than 10% above the average
	VerdictUnknown = "unknown" // no current price
)

// Deal is the buy/wait assessment for one price type.
type Deal struct {
	Type          string   `json:"type"`
	Label         string   `json:"label"`
	Verdict       string   `json:"verdict"`
	Reason        string   `json:"reason"`
	Current       *float64 `json:"current"`
	Lowest        *float64 `json:"lowest"`
	Average       *float64 `json:"average"`
	Highest       *float64 `json:"highest"`
	AboveLowPct   *float64 `json:"above_lowest_pct"`
	VsAveragePct  *float64 `json:"vs_average_pct"`
	PositionInRng *float64 `json:"position_in_range_pct"`
}

// Assess computes the verdict for one price row.
// Rules: buy = current <= lowest*1.03; good = current <= average*0.95;
// fair = current <= average*1.10; otherwise wait.
func Assess(p PriceStats) Deal {
	d := Deal{Type: p.Type, Label: p.Label, Current: p.Current, Lowest: p.Lowest, Average: p.Average, Highest: p.Highest}
	if p.Current == nil {
		d.Verdict = VerdictUnknown
		d.Reason = "no current price"
		if p.CurrentText != "" {
			d.Reason = "current price: " + p.CurrentText
		}
		return d
	}
	cur := *p.Current
	if p.Lowest != nil && *p.Lowest > 0 {
		d.AboveLowPct = ptr(round2((cur - *p.Lowest) / *p.Lowest * 100))
	}
	if p.Average != nil && *p.Average > 0 {
		d.VsAveragePct = ptr(round2((cur - *p.Average) / *p.Average * 100))
	}
	if p.Lowest != nil && p.Highest != nil && *p.Highest > *p.Lowest {
		d.PositionInRng = ptr(round2((cur - *p.Lowest) / (*p.Highest - *p.Lowest) * 100))
	}
	switch {
	case p.Lowest != nil && cur <= *p.Lowest*1.03:
		d.Verdict = VerdictBuy
		if cur <= *p.Lowest {
			d.Reason = "at the lowest price ever"
		} else {
			d.Reason = fmt.Sprintf("within %.1f%% of the lowest price ever", *d.AboveLowPct)
		}
	case p.Average != nil && *p.Average > 0 && cur <= *p.Average*0.95:
		d.Verdict = VerdictGood
		d.Reason = fmt.Sprintf("%.1f%% below the average price", -*d.VsAveragePct)
	case p.Average != nil && *p.Average > 0 && cur <= *p.Average*1.10:
		d.Verdict = VerdictFair
		if *d.VsAveragePct < 0 {
			d.Reason = fmt.Sprintf("%.1f%% below the average price", -*d.VsAveragePct)
		} else {
			d.Reason = fmt.Sprintf("%.1f%% above the average price", *d.VsAveragePct)
		}
	case p.Average != nil && *p.Average > 0:
		d.Verdict = VerdictWait
		d.Reason = fmt.Sprintf("%.1f%% above the average price", *d.VsAveragePct)
	default:
		d.Verdict = VerdictUnknown
		d.Reason = "no average price to compare with"
	}
	return d
}

// Best returns the deal with the lowest current price among assessed types,
// or false if none has a current price.
func Best(deals []Deal) (Deal, bool) {
	var best Deal
	found := false
	for _, d := range deals {
		if d.Current == nil {
			continue
		}
		if !found || *d.Current < *best.Current {
			best, found = d, true
		}
	}
	return best, found
}
