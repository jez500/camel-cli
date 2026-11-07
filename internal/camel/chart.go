// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // register PNG decoder
	"math"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Ranges supported by the chart service; RangeSince maps each to a start date.
var Ranges = []string{"1m", "3m", "6m", "1y", "all"}

// ValidRange reports whether r is a supported chart range.
func ValidRange(r string) bool {
	for _, x := range Ranges {
		if x == r {
			return true
		}
	}
	return false
}

// ChartOptions configures a chart image request.
type ChartOptions struct {
	Locale string
	ASIN   string
	Types  []string // amazon, new, used
	Range  string   // 1m, 3m, 6m, 1y, all
	Width  int
	Height int
	Legend bool
}

// ChartURL builds the charts.camelcamelcamel.com PNG URL.
func ChartURL(o ChartOptions) (string, error) {
	if !IsASIN(o.ASIN) {
		return "", fmt.Errorf("invalid ASIN %q", o.ASIN)
	}
	loc, err := LookupLocale(o.Locale)
	if err != nil {
		return "", err
	}
	if o.Range == "" {
		o.Range = "all"
	}
	if !ValidRange(o.Range) {
		return "", fmt.Errorf("invalid range %q (use one of %s)", o.Range, strings.Join(Ranges, ", "))
	}
	types, err := NormalizeTypes(o.Types)
	if err != nil {
		return "", err
	}
	if o.Width <= 0 {
		o.Width = DefaultChartWidth
	}
	if o.Height <= 0 {
		o.Height = DefaultChartHeight
	}
	o.Width = min(max(o.Width, 200), maxChartSide)
	o.Height = min(max(o.Height, 120), maxChartSide)
	q := url.Values{}
	q.Set("force", "1")
	q.Set("zero", "0")
	q.Set("w", fmt.Sprint(o.Width))
	q.Set("h", fmt.Sprint(o.Height))
	q.Set("desired", "false")
	if o.Legend {
		q.Set("legend", "1")
	} else {
		q.Set("legend", "0")
	}
	q.Set("ilt", "1")
	q.Set("tp", o.Range)
	q.Set("fo", "0")
	q.Set("lang", "en")
	return fmt.Sprintf("https://charts.camelcamelcamel.com/%s/%s/%s.png?%s", loc.Code, o.ASIN, strings.Join(types, "-"), q.Encode()), nil
}

// ValidType reports whether t is a price type (amazon, new, used).
func ValidType(t string) bool {
	for _, x := range PriceTypes {
		if x == t {
			return true
		}
	}
	return false
}

// NormalizeTypes validates and orders price types; empty means all three.
func NormalizeTypes(in []string) ([]string, error) {
	if len(in) == 0 {
		return append([]string(nil), PriceTypes...), nil
	}
	seen := map[string]bool{}
	for _, raw := range in {
		for _, t := range strings.Split(raw, ",") {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			if !ValidType(t) {
				return nil, fmt.Errorf("invalid price type %q (use %s)", t, strings.Join(PriceTypes, ", "))
			}
			seen[t] = true
		}
	}
	out := make([]string, 0, 3)
	for _, t := range PriceTypes {
		if seen[t] {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), PriceTypes...), nil
	}
	return out, nil
}

// Chart request defaults and limits. The service renders at 2x the request.
const (
	DefaultChartWidth  = 855
	DefaultChartHeight = 513
	maxChartSide       = 2000
	// maxDecodePixels caps image decoding (a 2000x2000 request renders 4000x4000).
	maxDecodePixels = 4000 * 4000
)

// Extraction tuning (pixels unless noted).
const (
	colorTolerance = 18    // RGB distance that still counts as the line color (anti-aliasing)
	maxRunHeight   = 8     // a horizontal price level is at most this thick; longer runs are vertical edges
	solidWindow    = 6     // columns checked on each side to tell solid from dotted lines
	solidRatio     = 0.85  // share of covered columns that makes a line solid
	spikeMargin    = 3     // a spike must reach this far beyond its neighbor levels
	spikeSearch    = 12    // columns searched each side for neighbor levels
	fitTolerance   = 0.03  // max axis-fit residual as a share of the price span
	snapTolerance  = 0.005 // values within this share of the price span snap to known prices
)

// Segment is one constant-price stretch read from the chart.
type Segment struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Price     float64 `json:"price"`
	Available bool    `json:"available"`
}

// Point is one day of a daily series.
type Point struct {
	Date      string  `json:"date"`
	Price     float64 `json:"price"`
	Available bool    `json:"available"`
}

// TypeCal holds the exact lifetime values for one price type.
type TypeCal struct {
	Highest float64
	Lowest  float64
	Current *float64
}

// Calibration ties chart pixels to real values. Start and End are the first
// and last days on the x axis of the combined "all" chart.
type Calibration struct {
	Types map[string]TypeCal
	Start time.Time
	End   time.Time
}

// Exact chart line colors (sampled from charts.camelcamelcamel.com).
var lineColors = map[string]color.RGBA{
	TypeAmazon: {99, 168, 94, 255},
	TypeNew:    {0, 51, 204, 255},
	TypeUsed:   {204, 51, 0, 255},
}

// ErrChartLayout means the image did not match the expected chart layout.
var ErrChartLayout = errors.New("chart layout not recognized")

func near(c color.Color, ref color.RGBA, tol int) bool {
	r, g, b, _ := c.RGBA()
	dr := int(r>>8) - int(ref.R)
	dg := int(g>>8) - int(ref.G)
	db := int(b>>8) - int(ref.B)
	return dr*dr+dg*dg+db*db <= tol*tol
}

func isDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return int(r>>8)+int(g>>8)+int(b>>8) < 200
}

type frame struct{ left, right, bottom int }

func findFrame(img image.Image) (frame, error) {
	b := img.Bounds()
	W, H := b.Dx(), b.Dy()
	at := func(x, y int) color.Color { return img.At(b.Min.X+x, b.Min.Y+y) }
	fr := frame{left: -1, right: -1, bottom: -1}
	for x := 0; x < W/4 && fr.left < 0; x++ {
		n := 0
		for y := H / 4; y < 3*H/4; y++ {
			if isDark(at(x, y)) {
				n++
			}
		}
		if n > H*4/10 {
			fr.left = x
		}
	}
	for y := H - 1; y > H/2 && fr.bottom < 0; y-- {
		n := 0
		for x := W / 4; x < 3*W/4; x++ {
			if isDark(at(x, y)) {
				n++
			}
		}
		if n > W*4/10 {
			fr.bottom = y
		}
	}
	if fr.left < 0 || fr.bottom < 0 {
		return fr, fmt.Errorf("%w: axes not found", ErrChartLayout)
	}
	for x := W - 1; x > fr.left; x-- {
		if isDark(at(x, fr.bottom)) {
			fr.right = x
			break
		}
	}
	if fr.right-fr.left < 50 {
		return fr, fmt.Errorf("%w: plot area too narrow", ErrChartLayout)
	}
	return fr, nil
}

type colRead struct {
	x     int
	y     float64
	solid bool
}

type colInfo struct{ top, bot int }

// readLine scans one color mask and returns per-column level readings, the
// per-column pixel extent, and the median line half-thickness.
func readLine(mask [][]bool, fr frame) ([]colRead, []colInfo, float64) {
	info := make([]colInfo, fr.right+2)
	reads := make([]colRead, 0, fr.right-fr.left)
	thick := make([]int, 0, 256)
	for x := fr.left + 1; x <= fr.right; x++ {
		col := mask[x]
		info[x] = colInfo{top: -1, bot: -1}
		var best []int
		for y := 0; y < fr.bottom; {
			if !col[y] {
				y++
				continue
			}
			start := y
			for y < fr.bottom && col[y] {
				y++
			}
			end := y - 1
			if info[x].top < 0 || start < info[x].top {
				info[x].top = start
			}
			if end > info[x].bot {
				info[x].bot = end
			}
			// Horizontal runs are at most maxRunHeight rows thick and continue sideways;
			// longer runs are vertical transitions.
			if end-start <= maxRunHeight && best == nil {
				if (x > fr.left+1 && anyIn(mask[x-1], start, end)) || (x < fr.right && anyIn(mask[x+1], start, end)) {
					best = []int{start, end}
				}
			}
		}
		if best == nil {
			continue
		}
		thick = append(thick, best[1]-best[0]+1)
		yc := float64(best[0]+best[1]) / 2
		row := int(math.Round(yc))
		// Solid vs dotted: a solid line covers nearly every column on at
		// least one side; a dotted line leaves gaps on both sides.
		side := func(dir int) float64 {
			hit, tot := 0, 0
			for d := 1; d <= solidWindow; d++ {
				xx := x + dir*d
				if xx <= fr.left || xx > fr.right {
					continue
				}
				tot++
				if anyIn(mask[xx], row-1, row+1) {
					hit++
				}
			}
			if tot == 0 {
				return 1
			}
			return float64(hit) / float64(tot)
		}
		reads = append(reads, colRead{x: x, y: yc, solid: math.Max(side(-1), side(1)) >= solidRatio})
	}
	half := 2.0
	if len(thick) > 0 {
		sort.Ints(thick)
		half = float64(thick[len(thick)/2]-1) / 2
	}
	// Fill transitions: a new level starts at the first gap column that
	// reaches it, and a gap that pokes beyond both neighbors is a spike.
	var extra []colRead
	for i := 0; i+1 < len(reads); i++ {
		a, b := reads[i], reads[i+1]
		if b.x-a.x < 2 {
			continue
		}
		for x := a.x + 1; x < b.x; x++ {
			if info[x].top >= 0 && float64(info[x].top) <= b.y+half && float64(info[x].bot) >= b.y-half {
				extra = append(extra, colRead{x: x, y: b.y, solid: b.solid})
				break
			}
		}
		top, bot := -1, -1
		for x := a.x + 1; x < b.x; x++ {
			if info[x].top < 0 {
				continue
			}
			if top < 0 || info[x].top < top {
				top = info[x].top
			}
			if info[x].bot > bot {
				bot = info[x].bot
			}
		}
		if top < 0 {
			continue
		}
		mid := (a.x + b.x) / 2
		if float64(top)+half < math.Min(a.y, b.y)-spikeMargin {
			extra = append(extra, colRead{x: mid, y: float64(top) + half, solid: true})
		}
		if float64(bot)-half > math.Max(a.y, b.y)+spikeMargin {
			extra = append(extra, colRead{x: mid, y: float64(bot) - half, solid: true})
		}
	}
	// Column-level spikes: a column whose line reaches beyond every level
	// read within spikeSearch columns on either side (spikes drawn over a level).
	levelAt := make(map[int]float64, len(reads))
	for _, r := range reads {
		levelAt[r.x] = r.y
	}
	for x := fr.left + 1; x <= fr.right; x++ {
		if info[x].top < 0 || info[x].bot-info[x].top <= maxRunHeight {
			continue
		}
		minL, maxL := math.Inf(1), math.Inf(-1)
		found := 0
		for _, dir := range []int{-1, 1} {
			for d := 1; d <= spikeSearch; d++ {
				if y, ok := levelAt[x+dir*d]; ok {
					minL, maxL = math.Min(minL, y), math.Max(maxL, y)
					found++
					break
				}
			}
		}
		if found == 0 {
			continue
		}
		if float64(info[x].top)+half < minL-spikeMargin {
			extra = append(extra, colRead{x: x, y: float64(info[x].top) + half, solid: true})
		}
		if float64(info[x].bot)-half > maxL+spikeMargin {
			extra = append(extra, colRead{x: x, y: float64(info[x].bot) - half, solid: true})
		}
	}
	if len(extra) > 0 {
		// A spike read replaces the level read in the same column.
		spikeX := map[int]bool{}
		for _, e := range extra {
			spikeX[e.x] = true
		}
		kept := reads[:0]
		for _, r := range reads {
			if !spikeX[r.x] {
				kept = append(kept, r)
			}
		}
		reads = kept
		reads = append(reads, extra...)
		sort.SliceStable(reads, func(i, j int) bool { return reads[i].x < reads[j].x })
	}
	return reads, info, half
}

// ExtractSeries reads the price lines of the requested types from the
// combined "all" chart (amazon-new-used) and returns constant-price
// segments per type. The price axis is fitted from each line's top and
// bottom pixel, which equal that type's exact lifetime highest and lowest.
// Values are approximate: dates are accurate to about one pixel column and
// prices to about one pixel row; values within 0.5% of the combined price
// span of a known lowest, highest or current price snap to it.
func ExtractSeries(png []byte, cal Calibration, types []string) (map[string][]Segment, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		return nil, fmt.Errorf("decoding chart: %w", err)
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return nil, fmt.Errorf("%w: image %dx%d is too large", ErrChartLayout, cfg.Width, cfg.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, fmt.Errorf("decoding chart: %w", err)
	}
	if !cal.End.After(cal.Start) {
		return nil, fmt.Errorf("%w: chart date range is empty", ErrChartLayout)
	}
	fr, err := findFrame(img)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()

	type lineData struct {
		reads []colRead
		half  float64
		top   int
		bot   int
	}
	lines := map[string]*lineData{}
	// Build masks and readings for every type that has calibration values,
	// so all of them contribute to the price-axis fit.
	for _, t := range PriceTypes {
		if _, ok := cal.Types[t]; !ok {
			continue
		}
		lc := lineColors[t]
		mask := make([][]bool, fr.right+2)
		any := false
		for x := fr.left + 1; x <= fr.right; x++ {
			col := make([]bool, fr.bottom)
			for y := 0; y < fr.bottom; y++ {
				if near(img.At(b.Min.X+x, b.Min.Y+y), lc, colorTolerance) {
					col[y] = true
					any = true
				}
			}
			mask[x] = col
		}
		if !any {
			continue
		}
		reads, info, half := readLine(mask, fr)
		top, bot := -1, -1
		for x := fr.left + 1; x <= fr.right; x++ {
			if info[x].top < 0 {
				continue
			}
			if top < 0 || info[x].top < top {
				top = info[x].top
			}
			if info[x].bot > bot {
				bot = info[x].bot
			}
		}
		lines[t] = &lineData{reads: reads, half: half, top: top, bot: bot}
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: no price lines found", ErrChartLayout)
	}

	// Fit price = a + k*y from (line top -> highest) and (line bottom -> lowest).
	type pt struct{ y, p float64 }
	var pts []pt
	gLo, gHi := math.Inf(1), math.Inf(-1)
	for t, ld := range lines {
		tc := cal.Types[t]
		pts = append(pts, pt{float64(ld.top) + ld.half, tc.Highest}, pt{float64(ld.bot) - ld.half, tc.Lowest})
		gLo, gHi = math.Min(gLo, tc.Lowest), math.Max(gHi, tc.Highest)
	}
	fit := func(ps []pt) (a, k, worst float64, wi int) {
		var sy, sp, syy, syp float64
		n := float64(len(ps))
		for _, q := range ps {
			sy += q.y
			sp += q.p
			syy += q.y * q.y
			syp += q.y * q.p
		}
		den := n*syy - sy*sy
		if den == 0 {
			return sp / n, 0, 0, -1
		}
		k = (n*syp - sy*sp) / den
		a = (sp - k*sy) / n
		wi = -1
		for i, q := range ps {
			if r := math.Abs(a + k*q.y - q.p); r > worst {
				worst, wi = r, i
			}
		}
		return a, k, worst, wi
	}
	span := math.Max(gHi-gLo, 0.01)
	a, k, worst, wi := fit(pts)
	// An extreme hidden behind another line breaks the fit: drop the worst
	// point while more than two remain.
	for worst > fitTolerance*span && len(pts) > 2 {
		pts = append(pts[:wi], pts[wi+1:]...)
		a, k, worst, wi = fit(pts)
	}
	flat := gHi-gLo < 0.005
	if !flat && (k >= 0 || worst > fitTolerance*span) {
		return nil, fmt.Errorf("%w: price axis could not be calibrated", ErrChartLayout)
	}
	price := func(y float64) float64 {
		if flat {
			return gHi
		}
		return a + k*y
	}

	daySpan := cal.End.Sub(cal.Start)
	dateAt := func(x int) string {
		f := float64(x-fr.left) / float64(fr.right-fr.left)
		return cal.Start.Add(time.Duration(f * float64(daySpan))).Truncate(24 * time.Hour).Format("2006-01-02")
	}

	out := map[string][]Segment{}
	for _, t := range types {
		ld, ok := lines[t]
		tc, okc := cal.Types[t]
		if !ok || !okc {
			continue
		}
		known := []float64{tc.Lowest, tc.Highest}
		if tc.Current != nil {
			known = append(known, *tc.Current)
		}
		tol := math.Max(snapTolerance*span, 0.01)
		snap := func(v float64) float64 {
			for _, kv := range known {
				if math.Abs(v-kv) <= tol {
					return kv
				}
			}
			if v < tc.Lowest {
				return tc.Lowest
			}
			if v > tc.Highest {
				return tc.Highest
			}
			return math.Round(v*100) / 100
		}
		var segs []Segment
		for _, r := range ld.reads {
			p := snap(price(r.y))
			d := dateAt(r.x)
			if n := len(segs); n > 0 {
				last := &segs[n-1]
				if math.Abs(last.Price-p) <= tol && last.Available == r.solid {
					last.To = d
					continue
				}
			}
			segs = append(segs, Segment{From: d, To: d, Price: p, Available: r.solid})
		}
		segs = mergeBlips(segs)
		if tc.Current != nil && len(segs) > 0 {
			last := &segs[len(segs)-1]
			end := cal.End.Format("2006-01-02")
			if math.Abs(last.Price-*tc.Current) > tol {
				segs = append(segs, Segment{From: end, To: end, Price: *tc.Current, Available: true})
			} else {
				last.To = end
			}
		}
		if segs == nil {
			segs = []Segment{}
		}
		out[t] = segs
	}
	return out, nil
}

func anyIn(col []bool, from, to int) bool {
	if col == nil {
		return false
	}
	if from < 0 {
		from = 0
	}
	for y := from; y <= to && y < len(col); y++ {
		if col[y] {
			return true
		}
	}
	return false
}

func mergeBlips(segs []Segment) []Segment {
	if len(segs) < 3 {
		return segs
	}
	out := make([]Segment, 0, len(segs))
	for i := 0; i < len(segs); i++ {
		s := segs[i]
		if s.From == s.To && i > 0 && i < len(segs)-1 && len(out) > 0 {
			prev := &out[len(out)-1]
			next := segs[i+1]
			if prev.Price == next.Price && prev.Available == next.Available {
				prev.To = next.To
				i++
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

// FilterSegments keeps the parts of segments on or after since (ISO date).
func FilterSegments(segs []Segment, since string) []Segment {
	if since == "" {
		return segs
	}
	out := make([]Segment, 0, len(segs))
	for _, s := range segs {
		if s.To < since {
			continue
		}
		if s.From < since {
			s.From = since
		}
		out = append(out, s)
	}
	return out
}

// Daily expands segments into one point per day, carrying each price
// forward until the next segment starts.
func Daily(segs []Segment) []Point {
	if len(segs) == 0 {
		return []Point{}
	}
	sorted := append([]Segment(nil), segs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].From < sorted[j].From })
	out := make([]Point, 0, 64)
	for i, s := range sorted {
		from, ok1 := ParseISODate(s.From)
		to, ok2 := ParseISODate(s.To)
		if !ok1 || !ok2 {
			continue
		}
		if i+1 < len(sorted) {
			if next, ok := ParseISODate(sorted[i+1].From); ok && next.After(to) {
				to = next.AddDate(0, 0, -1)
			}
		}
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			if n := len(out); n > 0 && out[n-1].Date >= ds {
				continue
			}
			out = append(out, Point{Date: ds, Price: s.Price, Available: s.Available})
		}
	}
	return out
}

// RangeSince returns the ISO start date for a range ending at end.
func RangeSince(r string, end time.Time) string {
	switch r {
	case "1m":
		return end.AddDate(0, -1, 0).Format("2006-01-02")
	case "3m":
		return end.AddDate(0, -3, 0).Format("2006-01-02")
	case "6m":
		return end.AddDate(0, -6, 0).Format("2006-01-02")
	case "1y":
		return end.AddDate(-1, 0, 0).Format("2006-01-02")
	}
	return ""
}
