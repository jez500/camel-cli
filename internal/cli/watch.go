// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
	"github.com/jez500/camel-cli/internal/cliutil"
)

func newWatchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Local watchlist: add products, then check them all for good prices (no camelcamelcamel account needed)",
		Long: strings.Trim(`
The watchlist lives in the local store. 'watch check' fetches every watched
product once (about one request per second), saves a snapshot, and prints a
verdict per product. 'lows' ranks the watchlist from those snapshots without
any network call.`, "\n"),
		Annotations: map[string]string{"pp:parent-group": "true", "pp:typed-exit-codes": "0,2"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newWatchAddCmd(flags), newWatchListCmd(flags), newWatchRemoveCmd(flags), newWatchCheckCmd(flags))
	return cmd
}

type watchAddResult struct {
	Input  string `json:"input"`
	ASIN   string `json:"asin,omitempty"`
	Locale string `json:"locale,omitempty"`
	Title  string `json:"title,omitempty"`
	Added  bool   `json:"added"`
	Note   string `json:"note,omitempty"`
	Error  string `json:"error,omitempty"`
}

// pp:data-source live
func newWatchAddCmd(flags *rootFlags) *cobra.Command {
	var locale string
	cmd := &cobra.Command{
		Use:   "add <amazon-url-or-asin>...",
		Short: "Add one or more products to the local watchlist (fetches each once to confirm it is tracked)",
		Example: strings.Trim(`
  camel-cli watch add https://www.amazon.com/dp/B07T32R43P
  camel-cli watch add B00OYGGMXK --locale au`, "\n"),
		Annotations: map[string]string{
			"pp:data-source":      "live",
			"pp:happy-args":       "input=B07T32R43P",
			"mcp:local-write":     "true",
			"pp:typed-exit-codes": "0,2,3,5,7",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "watch add")
			}
			if len(args) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass at least one Amazon URL or ASIN"))
			}
			if err := checkLocaleFlag(cmd, locale); err != nil {
				return err
			}
			ctx := cmd.Context()
			c := newCamelClient(flags)
			st, db, err := openCamelDB(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			results := make([]watchAddResult, 0, len(args))
			var lastErr error
			failed := 0
			for _, in := range args {
				rctx, rcancel := boundCtx(ctx, flags)
				t, err := resolveTarget(rctx, cmd, c, in, locale)
				rcancel()
				if err == nil {
					var p *camel.Product
					if p, err = fetchProduct(ctx, flags, c, t); err == nil {
						err = camel.SaveSnapshot(ctx, db, p, time.Now())
						if err == nil {
							added, aerr := camel.AddWatch(ctx, db, camel.WatchItem{ASIN: p.ASIN, Locale: p.Locale, Title: p.Title, AddedAt: time.Now().UTC().Format(time.RFC3339)})
							if aerr != nil {
								return aerr
							}
							r := watchAddResult{Input: in, ASIN: p.ASIN, Locale: p.Locale, Title: p.Title, Added: added}
							if !added {
								r.Note = "already on the watchlist"
							}
							results = append(results, r)
							continue
						}
					}
				}
				if isRateLimit(err) {
					return rateLimitErr(err)
				}
				failed++
				lastErr = err
				results = append(results, watchAddResult{Input: in, Error: err.Error()})
			}
			if err := emitWatchAdd(cmd, flags, results); err != nil {
				return err
			}
			if failed > 0 {
				if failed == len(args) {
					return camelErr(lastErr)
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d products were not added\n", failed, len(args))
			}
			return nil
		},
	}
	addLocaleFlag(cmd, &locale)
	return cmd
}

func emitWatchAdd(cmd *cobra.Command, flags *rootFlags, results []watchAddResult) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), results, flags)
	}
	for _, r := range results {
		switch {
		case r.Error != "":
			fmt.Fprintf(cmd.OutOrStdout(), "not added %s: %s\n", r.Input, r.Error)
		case r.Added:
			fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s): %s\n", r.ASIN, r.Locale, r.Title)
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "already watched %s (%s): %s\n", r.ASIN, r.Locale, r.Title)
		}
	}
	return nil
}

// pp:data-source local
func newWatchListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List every product on the local watchlist (ASIN, marketplace, date added, title); reads only the local store",
		Example:     "  camel-cli watch list --json",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "watch list")
			}
			items := make([]camel.WatchItem, 0)
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			st, db, ok, err := openCamelDBReadOnly(ctx)
			if err != nil {
				return err
			}
			if ok {
				defer st.Close()
				items, err = camel.Watchlist(ctx, db)
				if err != nil {
					return err
				}
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), items, flags)
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "The watchlist is empty. Add a product with: camel-cli watch add <amazon-url>")
				return nil
			}
			tw := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "ASIN\tLOCALE\tADDED\tTITLE")
			for _, it := range items {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.ASIN, it.Locale, it.AddedAt, truncate(it.Title, 70))
			}
			return tw.Flush()
		},
	}
	return cmd
}

// pp:data-source local
func newWatchRemoveCmd(flags *rootFlags) *cobra.Command {
	var locale string
	cmd := &cobra.Command{
		Use:         "remove <amazon-url-or-asin>",
		Short:       "Remove a product from the watchlist (snapshots stay for 'trend')",
		Example:     "  camel-cli watch remove B07T32R43P",
		Annotations: map[string]string{"pp:data-source": "local", "mcp:local-write": "true", "pp:happy-args": "input=B07T32R43P"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "watch remove")
			}
			if len(args) != 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass exactly one Amazon URL or ASIN"))
			}
			t, err := resolveLocalTarget(cmd, args[0], locale)
			if err != nil {
				return err
			}
			removed := false
			if camelDBExists() {
				ctx, cancel := boundCtx(cmd.Context(), flags)
				defer cancel()
				st, db, err := openCamelDB(ctx)
				if err != nil {
					return err
				}
				defer st.Close()
				removed, err = camel.RemoveWatch(ctx, db, t.ASIN, t.Locale)
				if err != nil {
					return err
				}
			}
			res := map[string]any{"asin": t.ASIN, "locale": t.Locale, "removed": removed}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), res, flags)
			}
			if removed {
				fmt.Fprintf(cmd.OutOrStdout(), "removed %s (%s)\n", t.ASIN, t.Locale)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (%s) was not on the watchlist\n", t.ASIN, t.Locale)
			}
			return nil
		},
	}
	addLocaleFlag(cmd, &locale)
	return cmd
}

type watchCheckRow struct {
	ASIN    string      `json:"asin"`
	Locale  string      `json:"locale"`
	Title   string      `json:"title,omitempty"`
	Verdict string      `json:"verdict"`
	Reason  string      `json:"reason"`
	Best    *camel.Deal `json:"best,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type watchCheckView struct {
	Checked       int             `json:"checked"`
	Rows          []watchCheckRow `json:"items"`
	FetchFailures []string        `json:"fetch_failures,omitempty"`
}

// pp:data-source live
func newWatchCheckCmd(flags *rootFlags) *cobra.Command {
	var only string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Fetch every watched product, save a snapshot, and print a buy/wait verdict for each",
		Long: strings.Trim(`
Fetches each watched product once (about one request per second) and saves a
snapshot. Use --only buy,good to print only products worth buying now. Use
'lows' afterwards to rank the watchlist offline.`, "\n"),
		Example: strings.Trim(`
  camel-cli watch check
  camel-cli watch check --only buy,good --json`, "\n"),
		Annotations: map[string]string{"pp:data-source": "live", "mcp:local-write": "true", "pp:typed-exit-codes": "0,5,7"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "watch check")
			}
			filter := map[string]bool{}
			for _, v := range splitCSV(only) {
				v = strings.ToLower(v)
				switch v {
				case camel.VerdictBuy, camel.VerdictGood, camel.VerdictFair, camel.VerdictWait, camel.VerdictUnknown:
					filter[v] = true
				default:
					return usageErr(fmt.Errorf("invalid --only value %q: use buy, good, fair, wait, unknown", v))
				}
			}
			view := watchCheckView{Rows: []watchCheckRow{}}
			if !camelDBExists() || cliutil.IsVerifyEnv() {
				return emitWatchCheck(cmd, flags, view)
			}
			ctx := cmd.Context()
			st, db, err := openCamelDB(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			items, err := camel.Watchlist(ctx, db)
			if err != nil {
				return err
			}
			if cliutil.IsDogfoodEnv() && len(items) > 2 {
				items = items[:2]
			}
			c := newCamelClient(flags)
			for _, it := range items {
				view.Checked++
				row := watchCheckRow{ASIN: it.ASIN, Locale: it.Locale, Title: it.Title}
				p, ferr := fetchProduct(ctx, flags, c, camel.Target{ASIN: it.ASIN, Locale: it.Locale})
				if ferr != nil {
					if isRateLimit(ferr) {
						return rateLimitErr(ferr)
					}
					row.Verdict, row.Error = camel.VerdictUnknown, ferr.Error()
					view.FetchFailures = append(view.FetchFailures, it.ASIN+"/"+it.Locale+": "+ferr.Error())
				} else {
					if err := camel.SaveSnapshot(ctx, db, p, time.Now()); err != nil {
						return err
					}
					row.Title = p.Title
					if best, ok := camel.Best(assessAll(p)); ok {
						row.Verdict, row.Reason, row.Best = best.Verdict, best.Label+": "+best.Reason, &best
					} else {
						row.Verdict, row.Reason = camel.VerdictUnknown, "no current price"
					}
				}
				if len(filter) > 0 && !filter[row.Verdict] {
					continue
				}
				view.Rows = append(view.Rows, row)
			}
			if n := len(view.FetchFailures); n > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d watched products could not be fetched\n", n, view.Checked)
				if n == view.Checked {
					_ = emitWatchCheck(cmd, flags, view)
					return apiErr(fmt.Errorf("all %d watched products failed to fetch: %s", n, view.FetchFailures[0]))
				}
			}
			return emitWatchCheck(cmd, flags, view)
		},
	}
	cmd.Flags().StringVar(&only, "only", "", "Only print these verdicts (comma list of buy, good, fair, wait, unknown)")
	return cmd
}

func emitWatchCheck(cmd *cobra.Command, flags *rootFlags, view watchCheckView) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), view, flags)
	}
	w := cmd.OutOrStdout()
	if view.Checked == 0 {
		fmt.Fprintln(w, "The watchlist is empty. Add a product with: camel-cli watch add <amazon-url>")
		return nil
	}
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "VERDICT\tASIN\tLOCALE\tPRICE\tREASON\tTITLE")
	for _, r := range view.Rows {
		price := "-"
		if r.Best != nil {
			price = money(r.Best.Current, r.Locale)
		}
		reason := r.Reason
		if r.Error != "" {
			reason = r.Error
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", strings.ToUpper(r.Verdict), r.ASIN, r.Locale, price, reason, truncate(r.Title, 50))
	}
	return tw.Flush()
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newWatchCmd(flags))
	})
}
