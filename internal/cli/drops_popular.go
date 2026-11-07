// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
	"github.com/jez500/camel-cli/internal/cliutil"
)

var dropPeriods = []string{"recent", "daily", "weekly", "relativedaily", "relativeweekly"}

type dropsOpts struct {
	locale, period, category, minDrop string
	minPct                            float64
	limit                             int
}

func addDropsFlags(cmd *cobra.Command, o *dropsOpts) {
	cmd.Flags().StringVar(&o.locale, "locale", "us", "Amazon marketplace: "+strings.Join(camel.LocaleCodes(), ", "))
	cmd.Flags().StringVar(&o.period, "period", "recent", "Drop window: "+strings.Join(dropPeriods, ", "))
	cmd.Flags().StringVar(&o.category, "category", "", "Amazon category slug, for example electronics, books, home-kitchen, toys-games")
	cmd.Flags().StringVar(&o.minDrop, "min-drop", "", "Site filter: minimum drop as a currency amount (1, 5, 10, 25, 50, 100) for recent/daily/weekly, or a percent for relative periods")
	cmd.Flags().Float64Var(&o.minPct, "min-pct", 0, "Keep only drops of at least this percent (applied locally)")
	cmd.Flags().IntVar(&o.limit, "limit", 0, "Maximum rows (0 = all the feed returns)")
}

func validPeriod(p string) bool {
	for _, x := range dropPeriods {
		if x == p {
			return true
		}
	}
	return false
}

func fetchDrops(ctx context.Context, flags *rootFlags, o dropsOpts) ([]camel.Drop, string, error) {
	if !validPeriod(o.period) {
		return nil, "", usageErr(fmt.Errorf("--period must be one of %s", strings.Join(dropPeriods, ", ")))
	}
	l, err := camel.LookupLocale(o.locale)
	if err != nil {
		return nil, "", usageErr(err)
	}
	host := "https://" + l.CamelHost
	q := url.Values{}
	q.Set("t", o.period)
	if o.category != "" {
		q.Set("bn", o.category)
	}
	if o.minDrop != "" {
		q.Set("d", o.minDrop)
	}
	resp, err := newCamelClient(flags).Get(ctx, host+"/top_drops/feed?"+q.Encode())
	if err != nil {
		return nil, l.Code, camelErr(err)
	}
	drops, err := camel.ParseDropsFeed(resp.Body)
	if err != nil {
		return nil, l.Code, apiErr(err)
	}
	out := make([]camel.Drop, 0, len(drops))
	for _, d := range drops {
		if o.minPct > 0 && (d.DropPct == nil || *d.DropPct < o.minPct) {
			continue
		}
		out = append(out, d)
	}
	if o.limit > 0 && len(out) > o.limit {
		out = out[:o.limit]
	}
	return out, l.Code, nil
}

func renderDrops(cmd *cobra.Command, locale string, drops []camel.Drop, seen []int) error {
	w := cmd.OutOrStdout()
	if len(drops) == 0 {
		fmt.Fprintln(w, "No price drops matched.")
		return nil
	}
	tw := newTabWriter(w)
	header := "DROP\tPRICE\tWAS\tASIN\tTITLE"
	if seen != nil {
		header += "\tSEEN BEFORE"
	}
	fmt.Fprintln(tw, header)
	for i, d := range drops {
		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s", pctPlain(d.DropPct), money(d.Price, locale), money(d.OldPrice, locale), d.ASIN, truncate(d.Title, 60))
		if seen != nil {
			line += fmt.Sprintf("\t%d", seen[i])
		}
		fmt.Fprintln(tw, line)
	}
	return tw.Flush()
}

func pctPlain(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("-%.1f%%", *p)
}

// configureDropsParent turns the generated `drops` group into a structured
// listing while keeping its generated `list` and `feed` subcommands.
func configureDropsParent(cmd *cobra.Command, flags *rootFlags) {
	var o dropsOpts
	addDropsFlags(cmd, &o)
	cmd.Short = "Top Amazon price drops with drop percent, price and previous price"
	cmd.Long = strings.Trim(`
Lists today's top price drops from the camelcamelcamel feed for a
marketplace. Use --period relativedaily for the biggest percentage drops.
The 'list' and 'feed' subcommands return the raw page and raw RSS.
Use 'fresh-drops' to hide drops you already saw in earlier pulls.`, "\n")
	cmd.Example = strings.Trim(`
  camel-cli drops --period daily --category electronics
  camel-cli drops --locale au --min-pct 30 --json`, "\n")
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	delete(cmd.Annotations, "pp:parent-group")
	cmd.Annotations["pp:data-source"] = "live"
	cmd.Annotations["mcp:read-only"] = "true"
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if dryRunOK(flags) {
			return writeDryRun(cmd.OutOrStdout(), flags, "drops")
		}
		if cliutil.IsVerifyEnv() {
			return printJSONFiltered(cmd.OutOrStdout(), []camel.Drop{}, flags)
		}
		ctx, cancel := boundCtx(cmd.Context(), flags)
		defer cancel()
		drops, loc, err := fetchDrops(ctx, flags, o)
		if err != nil {
			return err
		}
		if !wantsHumanTable(cmd.OutOrStdout(), flags) {
			return printJSONFiltered(cmd.OutOrStdout(), drops, flags)
		}
		return renderDrops(cmd, loc, drops, nil)
	}
}

// configurePopularParent turns the generated `popular` group into a
// structured listing while keeping its generated subcommands.
func configurePopularParent(cmd *cobra.Command, flags *rootFlags) {
	var locale string
	var dealsOnly bool
	var minBelow float64
	var limit int
	cmd.Flags().StringVar(&locale, "locale", "us", "Amazon marketplace: "+strings.Join(camel.LocaleCodes(), ", "))
	cmd.Flags().BoolVar(&dealsOnly, "deals-only", false, "Only products camelcamelcamel marks as a deal")
	cmd.Flags().Float64Var(&minBelow, "min-below-avg", 0, "Keep only items at least this percent below their average price")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows (0 = all the feed returns)")
	cmd.Short = "Popular Amazon products with current, list and average price"
	cmd.Example = strings.Trim(`
  camel-cli popular --deals-only
  camel-cli popular --min-below-avg 20 --json`, "\n")
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	delete(cmd.Annotations, "pp:parent-group")
	cmd.Annotations["pp:data-source"] = "live"
	cmd.Annotations["mcp:read-only"] = "true"
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if dryRunOK(flags) {
			return writeDryRun(cmd.OutOrStdout(), flags, "popular")
		}
		if cliutil.IsVerifyEnv() {
			return printJSONFiltered(cmd.OutOrStdout(), []camel.PopularItem{}, flags)
		}
		l, err := camel.LookupLocale(locale)
		if err != nil {
			return usageErr(err)
		}
		host := "https://" + l.CamelHost
		ctx, cancel := boundCtx(cmd.Context(), flags)
		defer cancel()
		deal := "0"
		if dealsOnly {
			deal = "1"
		}
		resp, err := newCamelClient(flags).Get(ctx, host+"/popular.xml?deal="+deal)
		if err != nil {
			return camelErr(err)
		}
		items, err := camel.ParsePopularFeed(resp.Body)
		if err != nil {
			return apiErr(err)
		}
		out := make([]camel.PopularItem, 0, len(items))
		for _, it := range items {
			if minBelow > 0 && (it.BelowAvgPct == nil || *it.BelowAvgPct < minBelow) {
				continue
			}
			out = append(out, it)
		}
		if limit > 0 && len(out) > limit {
			out = out[:limit]
		}
		if !wantsHumanTable(cmd.OutOrStdout(), flags) {
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		}
		w := cmd.OutOrStdout()
		if len(out) == 0 {
			fmt.Fprintln(w, "No popular products matched.")
			return nil
		}
		tw := newTabWriter(w)
		fmt.Fprintln(tw, "CURRENT\tAVERAGE\tBELOW AVG\tLIST\tASIN\tTITLE")
		for _, it := range out {
			below := "-"
			if it.BelowAvgPct != nil {
				below = fmt.Sprintf("%.1f%%", *it.BelowAvgPct)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", money(it.Current, l.Code), money(it.Average, l.Code), below, money(it.ListPrice, l.Code), it.ASIN, truncate(it.Title, 60))
		}
		return tw.Flush()
	}
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		if c, _, err := root.Find([]string{"drops"}); err == nil && c != root {
			configureDropsParent(c, flags)
		}
		if c, _, err := root.Find([]string{"popular"}); err == nil && c != root {
			configurePopularParent(c, flags)
		}
	})
}
