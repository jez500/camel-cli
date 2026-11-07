// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type compareRow struct {
	Locale      string       `json:"locale"`
	Marketplace string       `json:"marketplace"`
	Currency    string       `json:"currency"`
	Tracked     bool         `json:"tracked"`
	Title       string       `json:"title,omitempty"`
	URL         string       `json:"url,omitempty"`
	Deals       []camel.Deal `json:"deals"`
	Error       string       `json:"error,omitempty"`
}

type compareView struct {
	ASIN          string       `json:"asin"`
	Rows          []compareRow `json:"marketplaces"`
	Note          string       `json:"note"`
	FetchFailures []string     `json:"fetch_failures,omitempty"`
}

func newNovelCompareCmd(flags *rootFlags) *cobra.Command {
	var flagLocales, locale string
	cmd := &cobra.Command{
		Use:   "compare <amazon-url-or-asin>",
		Short: "See one product's current, lowest and average prices on several Amazon marketplaces side by side",
		Long: strings.Trim(`
Use this command to see one product's prices on several Amazon marketplaces
side by side (for example US and Australia). Prices stay in each
marketplace's own currency; camel-cli does not convert currencies. A
marketplace where camelcamelcamel does not track the ASIN shows as not
tracked. Do NOT use it for a single marketplace; use 'history' instead.`, "\n"),
		Example: strings.Trim(`
  camel-cli compare https://www.amazon.com.au/dp/B0BSHF7WHW --locales us,au
  camel-cli compare B07T32R43P --locales us,uk,ca --agent`, "\n"),
		Annotations: map[string]string{
			"mcp:local-write":     "true",
			"pp:data-source":      "live",
			"pp:happy-args":       "input=B07T32R43P;--locales=us,au",
			"pp:typed-exit-codes": "0,2,5,7",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "compare")
			}
			if len(args) != 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass exactly one Amazon URL or ASIN"))
			}
			codes := splitCSV(flagLocales)
			if len(codes) < 2 {
				return usageErr(fmt.Errorf("--locales needs at least two marketplaces, for example us,au"))
			}
			locs := make([]camel.Locale, 0, len(codes))
			for _, code := range codes {
				l, lerr := camel.LookupLocale(code)
				if lerr != nil {
					return usageErr(lerr)
				}
				locs = append(locs, l)
			}
			ctx := cmd.Context()
			c := newCamelClient(flags)
			rctx, rcancel := boundCtx(ctx, flags)
			t, err := resolveTarget(rctx, cmd, c, args[0], locale)
			rcancel()
			if err != nil {
				return err
			}
			view := compareView{ASIN: t.ASIN, Rows: []compareRow{}, Note: "Prices are in each marketplace's own currency; no conversion."}
			for _, l := range locs {
				row := compareRow{Locale: l.Code, Marketplace: l.AmazonDomain, Currency: l.Currency, Deals: []camel.Deal{}}
				p, ferr := fetchProduct(ctx, flags, c, camel.Target{ASIN: t.ASIN, Locale: l.Code})
				switch {
				case ferr == nil:
					row.Tracked, row.Title, row.URL, row.Deals = true, p.Title, p.URL, assessAll(p)
					saveSnapshot(ctx, cmd.ErrOrStderr(), p)
				case errors.Is(ferr, camel.ErrNotTracked):
					row.Error = "not tracked on " + l.CamelHost
				default:
					if isRateLimit(ferr) {
						return rateLimitErr(ferr)
					}
					row.Error = ferr.Error()
					view.FetchFailures = append(view.FetchFailures, l.Code+": "+ferr.Error())
				}
				view.Rows = append(view.Rows, row)
			}
			if n := len(view.FetchFailures); n > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d marketplace fetches failed\n", n, len(codes))
				if n == len(codes) {
					_ = emitCompare(cmd, flags, view)
					return apiErr(fmt.Errorf("all %d marketplace fetches failed: %s", n, view.FetchFailures[0]))
				}
			}
			return emitCompare(cmd, flags, view)
		},
	}
	cmd.Flags().StringVar(&flagLocales, "locales", "us,au", "Marketplaces to compare, comma separated: "+strings.Join(camel.LocaleCodes(), ", "))
	addLocaleFlag(cmd, &locale)
	return cmd
}

func emitCompare(cmd *cobra.Command, flags *rootFlags, view compareView) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), view, flags)
	}
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ASIN %s across marketplaces (local currencies, no conversion)\n\n", view.ASIN)
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "MARKET\tPRICE TYPE\tCURRENT\tLOWEST\tAVERAGE\tVS LOWEST\tVERDICT")
	for _, r := range view.Rows {
		if !r.Tracked {
			fmt.Fprintf(tw, "%s\t-\t-\t-\t-\t-\t%s\n", r.Locale, r.Error)
			continue
		}
		for _, d := range r.Deals {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.Locale, d.Label, money(d.Current, r.Locale), money(d.Lowest, r.Locale), money(d.Average, r.Locale), pct(d.AboveLowPct), strings.ToUpper(d.Verdict))
		}
	}
	return tw.Flush()
}
