// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source local

package cli

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type trendRow struct {
	FetchedAt  string   `json:"fetched_at"`
	Type       string   `json:"type"`
	Current    *float64 `json:"current"`
	Change     *float64 `json:"change"`
	Lowest     *float64 `json:"lowest"`
	NewLow     bool     `json:"new_all_time_low"`
	LowestDate string   `json:"lowest_date,omitempty"`
}

type trendView struct {
	ASIN      string     `json:"asin"`
	Locale    string     `json:"locale"`
	Title     string     `json:"title,omitempty"`
	Snapshots int        `json:"snapshots"`
	Rows      []trendRow `json:"rows"`
	Note      string     `json:"note,omitempty"`
}

func newNovelTrendCmd(flags *rootFlags) *cobra.Command {
	var flagType, locale string
	cmd := &cobra.Command{
		Use:   "trend <amazon-url-or-asin>",
		Short: "See how a product's price moved across your own past lookups, including new all-time lows",
		Long: strings.Trim(`
Use this command to see how one product's price changed across your own past
lookups, read from the local store with no network call. Every 'history',
'deal', 'compare', 'watch add' and 'watch check' saves a snapshot. A row is marked as a new
all-time low when the lowest price or its date changed since the previous
snapshot. Do NOT use it for the current camelcamelcamel summary; use
'history' instead.`, "\n"),
		Example: strings.Trim(`
  camel-cli trend B07T32R43P --type amazon
  camel-cli trend https://www.amazon.com.au/dp/B00OYGGMXK --agent`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "input=B07T32R43P",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "trend")
			}
			if len(args) != 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass exactly one Amazon URL or ASIN"))
			}
			t, err := resolveLocalTarget(cmd, args[0], locale)
			if err != nil {
				return err
			}
			priceType, err := parseSingleType(flagType)
			if err != nil {
				return err
			}
			view := trendView{ASIN: t.ASIN, Locale: t.Locale, Rows: []trendRow{}}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			st, db, ok, err := openCamelDBReadOnly(ctx)
			if err != nil {
				return err
			}
			if !ok {
				view.Note = "no snapshots yet; run: camel-cli history " + t.ASIN
				return emitTrend(cmd, flags, view)
			}
			defer st.Close()
			snaps, err := camel.Snapshots(ctx, db, t.ASIN, t.Locale, priceType)
			if err != nil {
				return err
			}
			view.Snapshots = len(snaps)
			prev := map[string]camel.Snapshot{}
			for _, s := range snaps {
				if view.Title == "" {
					view.Title = s.Title
				}
				r := trendRow{FetchedAt: s.FetchedAt, Type: s.Type, Current: s.Current, Lowest: s.Lowest, LowestDate: s.LowestDate}
				if p, ok := prev[s.Type]; ok {
					if p.Current != nil && s.Current != nil {
						d := math.Round((*s.Current-*p.Current)*100) / 100
						r.Change = &d
					}
					if s.Lowest != nil && p.Lowest != nil && (*s.Lowest < *p.Lowest || s.LowestDate != p.LowestDate) {
						r.NewLow = true
					}
				}
				prev[s.Type] = s
				view.Rows = append(view.Rows, r)
			}
			if n := len(snaps); n > 0 {
				if newest, perr := time.Parse(time.RFC3339, snaps[n-1].FetchedAt); perr == nil {
					staleHint(cmd, flags, newest, "camel-cli history "+view.ASIN)
				}
			}
			if len(snaps) == 0 {
				view.Note = "no snapshots for this product; run: camel-cli history " + t.ASIN
			}
			return emitTrend(cmd, flags, view)
		},
	}
	cmd.Flags().StringVar(&flagType, "type", "", "Only one price type: amazon, new, used (default all)")
	addLocaleFlag(cmd, &locale)
	return cmd
}

func emitTrend(cmd *cobra.Command, flags *rootFlags, view trendView) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), view, flags)
	}
	w := cmd.OutOrStdout()
	if len(view.Rows) == 0 {
		fmt.Fprintln(w, view.Note)
		return nil
	}
	fmt.Fprintf(w, "%s (%s, %s): %d snapshots\n\n", view.Title, view.ASIN, view.Locale, view.Snapshots)
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "FETCHED\tTYPE\tCURRENT\tCHANGE\tLOWEST EVER\t")
	for _, r := range view.Rows {
		ch := "-"
		if r.Change != nil {
			ch = fmt.Sprintf("%+.2f", *r.Change)
		}
		mark := ""
		if r.NewLow {
			mark = "new all-time low"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", r.FetchedAt, r.Type, money(r.Current, view.Locale), ch, withDate(r.Lowest, r.LowestDate, view.Locale), mark)
	}
	return tw.Flush()
}
