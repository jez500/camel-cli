// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type dealView struct {
	ASIN     string       `json:"asin"`
	Locale   string       `json:"locale"`
	Title    string       `json:"title"`
	URL      string       `json:"url"`
	Verdict  string       `json:"verdict"`
	Reason   string       `json:"reason"`
	BestType string       `json:"best_type,omitempty"`
	Deals    []camel.Deal `json:"deals"`
}

func newDealCmd(flags *rootFlags) *cobra.Command {
	var locale string
	cmd := &cobra.Command{
		Use:   "deal <amazon-url-or-asin>",
		Short: "Buy-or-wait verdict: current price vs the lowest and average price for each price type",
		Long: strings.Trim(`
Use this command for a quick buy/wait answer about one product. It makes one
request (no chart) and compares each current price with that price type's
lowest-ever and average price:

  buy   current is at or within 3% of the lowest price ever
  good  current is at least 5% below the average price
  fair  current is from 5% below to 10% above the average price
  wait  current is more than 10% above the average price

The headline verdict uses the cheapest price type that has a current price.
Do NOT use this command for dated price changes; use 'history' instead.`, "\n"),
		Example: strings.Trim(`
  camel-cli deal https://www.amazon.com/dp/B07T32R43P
  camel-cli deal B00OYGGMXK --locale au --json`, "\n"),
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
				return writeDryRun(cmd.OutOrStdout(), flags, "deal")
			}
			if len(args) != 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass exactly one Amazon URL or ASIN"))
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
			view := dealView{ASIN: p.ASIN, Locale: p.Locale, Title: p.Title, URL: p.URL, Deals: assessAll(p)}
			if best, ok := camel.Best(view.Deals); ok {
				view.Verdict, view.Reason, view.BestType = best.Verdict, best.Label+": "+best.Reason, best.Type
			} else {
				view.Verdict, view.Reason = camel.VerdictUnknown, "no current price for any price type"
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			renderProductHeader(w, p)
			fmt.Fprintf(w, "\nVerdict: %s  %s\n\n", verdictWord(view.Verdict), view.Reason)
			tw := newTabWriter(w)
			fmt.Fprintln(tw, "PRICE TYPE\tVERDICT\tCURRENT\tLOWEST\tAVERAGE\tVS LOWEST\tVS AVERAGE")
			for _, d := range view.Deals {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", d.Label, strings.ToUpper(d.Verdict), money(d.Current, p.Locale), money(d.Lowest, p.Locale), money(d.Average, p.Locale), pct(d.AboveLowPct), pct(d.VsAveragePct))
			}
			return tw.Flush()
		},
	}
	addLocaleFlag(cmd, &locale)
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newDealCmd(flags))
	})
}
