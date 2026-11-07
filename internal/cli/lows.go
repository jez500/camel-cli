// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source local

package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type lowsRow struct {
	ASIN        string   `json:"asin"`
	Locale      string   `json:"locale"`
	Title       string   `json:"title,omitempty"`
	Type        string   `json:"type"`
	Current     *float64 `json:"current"`
	Lowest      *float64 `json:"lowest"`
	LowestDate  string   `json:"lowest_date,omitempty"`
	Average     *float64 `json:"average"`
	AboveLowPct float64  `json:"above_lowest_pct"`
	VsAvgPct    *float64 `json:"vs_average_pct"`
	NearLow     bool     `json:"near_low"`
	FetchedAt   string   `json:"snapshot_at"`
}

func newNovelLowsCmd(flags *rootFlags) *cobra.Command {
	var within float64
	var flagType string
	var limit int
	cmd := &cobra.Command{
		Use:   "lows",
		Short: "Rank watched products by how close their current price is to their lowest-ever price",
		Long: strings.Trim(`
Use this command to rank all watched products by distance from their
lowest-ever price, using stored snapshots with no network call. Rows within
--within percent of the low are flagged near_low. Do NOT use it to refresh
prices or fire alerts; use 'watch check' instead. Do NOT use it for one
product's verdict; use 'deal' instead.`, "\n"),
		Example: strings.Trim(`
  camel-cli lows --within 10
  camel-cli lows --type amazon --agent`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "lows")
			}
			if within < 0 {
				return usageErr(fmt.Errorf("--within must be 0 or more"))
			}
			priceType, err := parseSingleType(flagType)
			if err != nil {
				return err
			}
			rows := make([]lowsRow, 0)
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			st, db, ok, err := openCamelDBReadOnly(ctx)
			if err != nil {
				return err
			}
			if ok {
				defer st.Close()
				snaps, err := camel.LatestSnapshots(ctx, db)
				if err != nil {
					return err
				}
				for _, s := range snaps {
					if priceType != "" && s.Type != priceType {
						continue
					}
					if s.Current == nil || s.Lowest == nil || *s.Lowest <= 0 {
						continue
					}
					r := lowsRow{ASIN: s.ASIN, Locale: s.Locale, Title: s.Title, Type: s.Type, Current: s.Current, Lowest: s.Lowest, LowestDate: s.LowestDate, Average: s.Average, FetchedAt: s.FetchedAt}
					d := camel.Assess(camel.PriceStats{Current: s.Current, Lowest: s.Lowest, Average: s.Average})
					if d.AboveLowPct != nil {
						r.AboveLowPct = *d.AboveLowPct
					}
					r.VsAvgPct = d.VsAveragePct
					r.NearLow = r.AboveLowPct <= within
					rows = append(rows, r)
				}
			}
			newest := time.Time{}
			for _, r := range rows {
				if t, perr := time.Parse(time.RFC3339, r.FetchedAt); perr == nil && t.After(newest) {
					newest = t
				}
			}
			staleHint(cmd, flags, newest, "camel-cli watch check")
			sort.SliceStable(rows, func(i, j int) bool { return rows[i].AboveLowPct < rows[j].AboveLowPct })
			if limit > 0 && len(rows) > limit {
				rows = rows[:limit]
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				if len(rows) == 0 {
					fmt.Fprintln(cmd.ErrOrStderr(), "hint: no watched products with snapshots; run 'camel-cli watch add <amazon-url>' then 'watch check'")
				}
				return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
			}
			w := cmd.OutOrStdout()
			if len(rows) == 0 {
				fmt.Fprintln(w, "No watched products with snapshots. Add some with 'watch add', then run 'watch check'.")
				return nil
			}
			tw := newTabWriter(w)
			fmt.Fprintln(tw, "ABOVE LOW\tASIN\tLOCALE\tTYPE\tCURRENT\tLOWEST EVER\tVS AVG\tTITLE")
			for _, r := range rows {
				flag := ""
				if r.NearLow {
					flag = " *"
				}
				fmt.Fprintf(tw, "%+.1f%%%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.AboveLowPct, flag, r.ASIN, r.Locale, r.Type, money(r.Current, r.Locale), withDate(r.Lowest, r.LowestDate, r.Locale), pct(r.VsAvgPct), truncate(r.Title, 40))
			}
			if err := tw.Flush(); err != nil {
				return err
			}
			fmt.Fprintf(w, "\n* within %.0f%% of the lowest price ever (from the latest snapshot; run 'watch check' to refresh)\n", within)
			return nil
		},
	}
	cmd.Flags().Float64Var(&within, "within", 10, "Flag products within this percent of their lowest-ever price")
	cmd.Flags().StringVar(&flagType, "type", "", "Only one price type: amazon, new, used (default all)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows (0 = all)")
	return cmd
}
