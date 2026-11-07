// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type seriesView struct {
	Segments []camel.Segment `json:"segments"`
	Points   []camel.Point   `json:"points,omitempty"`
}

type historyView struct {
	Product     *camel.Product        `json:"product"`
	Verdicts    []camel.Deal          `json:"verdicts"`
	Range       string                `json:"range"`
	Series      map[string]seriesView `json:"series"`
	SeriesNote  string                `json:"series_note,omitempty"`
	SeriesError string                `json:"series_error,omitempty"`
}

func newHistoryCmd(flags *rootFlags) *cobra.Command {
	var locale, typeFlag, rng string
	var daily, noSeries bool
	cmd := &cobra.Command{
		Use:   "history <amazon-url-or-asin>",
		Short: "Price history for an Amazon product: lowest, highest, current, average, and dated price changes",
		Long: strings.Trim(`
Give an Amazon product URL (any form: /dp/, /gp/product/, a.co or amzn.to
short links, any supported marketplace domain) or a bare ASIN. camel-cli
reads the camelcamelcamel page for that marketplace and prints:

  - the exact summary for each price type (Amazon, 3rd Party New, 3rd Party
    Used): lowest and highest ever with dates, current price, average
  - a buy/wait verdict per price type
  - the dated price changes, read from the camelcamelcamel chart image
    (approximate: about one day per pixel column; see series_note)

Each lookup also saves a local snapshot, which 'trend' and 'lows' read.`, "\n"),
		Example: strings.Trim(`
  camel-cli history https://www.amazon.com/dp/B07T32R43P
  camel-cli history 'https://www.amazon.com.au/dp/B00OYGGMXK' --range 1y
  camel-cli history B07T32R43P --type all --daily --json
  camel-cli history B07T32R43P --no-series --agent --select product.title,verdicts`, "\n"),
		Annotations: map[string]string{
			"mcp:local-write":     "true",
			"pp:data-source":      "live",
			"pp:happy-args":       "input=B07T32R43P",
			"pp:typed-exit-codes": "0,2,3,5,7",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "history")
			}
			if len(args) != 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass exactly one Amazon URL or ASIN"))
			}
			if err := validateRange(rng); err != nil {
				return err
			}
			if tf := strings.ToLower(strings.TrimSpace(typeFlag)); tf != "" && tf != "all" {
				if _, err := camel.NormalizeTypes(splitCSV(typeFlag)); err != nil {
					return usageErr(err)
				}
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c := newCamelClient(flags)
			t, err := resolveTarget(ctx, cmd, c, args[0], locale)
			if err != nil {
				return err
			}
			p, err := c.FetchProduct(ctx, t)
			if err != nil {
				return camelErr(err)
			}
			saveSnapshot(ctx, cmd.ErrOrStderr(), p)
			view := historyView{Product: p, Range: rng, Series: map[string]seriesView{}, Verdicts: assessAll(p)}
			if !noSeries {
				types, err := pickSeriesTypes(p, typeFlag)
				if err != nil {
					return err
				}
				if len(types) > 0 {
					all, serr := c.Series(ctx, p, types, time.Now())
					if serr != nil {
						view.SeriesError = serr.Error()
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: price series unavailable: %v\n", serr)
					} else {
						end, _ := camel.ParseISODate(p.LastChecked)
						if end.IsZero() {
							end = time.Now().UTC()
						}
						since := camel.RangeSince(rng, end)
						for _, typ := range types {
							segs := camel.FilterSegments(all[typ], since)
							if segs == nil {
								segs = []camel.Segment{}
							}
							sv := seriesView{Segments: segs}
							if daily {
								sv.Points = camel.Daily(segs)
							}
							view.Series[typ] = sv
						}
						view.SeriesNote = camel.SeriesNote
					}
				}
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderHistory(cmd.OutOrStdout(), view, daily)
			return nil
		},
	}
	addLocaleFlag(cmd, &locale)
	cmd.Flags().StringVar(&typeFlag, "type", "", "Price types for the dated series: amazon, new, used, a comma list, or all (default: Amazon, else the first type with data)")
	cmd.Flags().StringVar(&rng, "range", "all", "Series time range: 1m, 3m, 6m, 1y, all")
	cmd.Flags().BoolVar(&daily, "daily", false, "Expand the series to one row per day")
	cmd.Flags().BoolVar(&noSeries, "no-series", false, "Skip the chart download; print only the summary and verdicts")
	return cmd
}

func renderProductHeader(w io.Writer, p *camel.Product) {
	l, _ := camel.LookupLocale(p.Locale)
	fmt.Fprintln(w, bold(p.Title))
	parts := []string{"ASIN " + p.ASIN, l.AmazonDomain + " (" + p.Locale + ")"}
	if p.TrackingSince != "" {
		parts = append(parts, "tracked since "+p.TrackingSince)
	}
	if p.LastChecked != "" {
		parts = append(parts, "checked "+p.LastChecked)
	}
	fmt.Fprintln(w, strings.Join(parts, " · "))
	fmt.Fprintln(w, p.URL)
}

func renderSummary(w io.Writer, p *camel.Product) {
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "PRICE TYPE\tLOWEST EVER\tHIGHEST EVER\tCURRENT\tAVERAGE")
	for _, ps := range p.Prices {
		cur := withDate(ps.Current, ps.CurrentDate, p.Locale)
		if ps.Current == nil && ps.CurrentText != "" {
			cur = ps.CurrentText
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", ps.Label,
			withDate(ps.Lowest, ps.LowestDate, p.Locale), withDate(ps.Highest, ps.HighestDate, p.Locale), cur, money(ps.Average, p.Locale))
	}
	_ = tw.Flush()
}

func verdictWord(v string) string {
	switch v {
	case camel.VerdictBuy:
		return green("BUY")
	case camel.VerdictGood:
		return green("GOOD")
	case camel.VerdictFair:
		return yellow("FAIR")
	case camel.VerdictWait:
		return red("WAIT")
	}
	return "UNKNOWN"
}

func renderHistory(w io.Writer, v historyView, daily bool) {
	p := v.Product
	renderProductHeader(w, p)
	fmt.Fprintln(w)
	renderSummary(w, p)
	if len(v.Verdicts) > 0 {
		fmt.Fprintln(w)
		for _, d := range v.Verdicts {
			fmt.Fprintf(w, "%-15s %s  %s\n", d.Label+":", verdictWord(d.Verdict), d.Reason)
		}
	}
	if v.SeriesError != "" {
		fmt.Fprintf(w, "\nPrice series unavailable: %s\n", v.SeriesError)
		return
	}
	for _, typ := range camel.PriceTypes {
		sv, ok := v.Series[typ]
		if !ok {
			continue
		}
		rangeLabel := "all time"
		if v.Range != "all" {
			rangeLabel = "last " + v.Range
		}
		fmt.Fprintf(w, "\n%s price history, %s (approximate, read from the chart)\n", camel.PriceTypeLabel(typ), rangeLabel)
		tw := newTabWriter(w)
		if daily {
			fmt.Fprintln(tw, "DATE\tPRICE\t")
			for _, pt := range sv.Points {
				note := ""
				if !pt.Available {
					note = "not available"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", pt.Date, moneyV(pt.Price, p.Locale), note)
			}
		} else {
			fmt.Fprintln(tw, "FROM\tTO\tPRICE\t")
			for _, s := range sv.Segments {
				note := ""
				if !s.Available {
					note = "not available (last known price)"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.From, s.To, moneyV(s.Price, p.Locale), note)
			}
		}
		_ = tw.Flush()
	}
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newHistoryCmd(flags))
	})
}
