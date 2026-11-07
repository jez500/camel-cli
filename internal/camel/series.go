// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"context"
	"fmt"
	"time"
)

// SeriesNote explains how the series is produced. Commands include it in output.
const SeriesNote = "Approximate: read from the camelcamelcamel 'all' chart image and calibrated on the exact lowest and highest prices. Dates are accurate to about one day per pixel column; short spikes can be off by a few days or cents. available=false marks dotted stretches where the price was not available."

// Series chart request size: the largest allowed width (maxChartSide) gives
// the most pixel columns per day; the service renders at 2x.
const (
	chartWidth  = 2000
	chartHeight = 800
)

// CalibrationFor builds the chart calibration from the product page.
func CalibrationFor(p *Product, now time.Time) (Calibration, error) {
	start, ok := ParseISODate(p.TrackingSince)
	if !ok {
		return Calibration{}, fmt.Errorf("tracking start date not found on the product page")
	}
	end := now.UTC().Truncate(24 * time.Hour)
	if lc, ok := ParseISODate(p.LastChecked); ok {
		end = lc
	}
	cal := Calibration{Types: map[string]TypeCal{}, Start: start, End: end}
	for _, ps := range p.Prices {
		if ps.Lowest != nil && ps.Highest != nil {
			cal.Types[ps.Type] = TypeCal{Highest: *ps.Highest, Lowest: *ps.Lowest, Current: ps.Current}
		}
	}
	if len(cal.Types) == 0 {
		return Calibration{}, fmt.Errorf("no price history on this page")
	}
	return cal, nil
}

// Series downloads the combined "all" chart once and extracts the
// requested price types. Types without history are left out of the map.
func (c *Client) Series(ctx context.Context, p *Product, types []string, now time.Time) (map[string][]Segment, error) {
	cal, err := CalibrationFor(p, now)
	if err != nil {
		return nil, err
	}
	png, _, err := c.FetchChart(ctx, ChartOptions{Locale: p.Locale, ASIN: p.ASIN, Types: PriceTypes, Range: "all", Width: chartWidth, Height: chartHeight})
	if err != nil {
		return nil, err
	}
	return ExtractSeries(png, cal, types)
}
