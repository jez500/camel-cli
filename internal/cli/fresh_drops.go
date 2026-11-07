// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
	"github.com/jez500/camel-cli/internal/cliutil"
)

type freshDrop struct {
	camel.Drop
	SeenBefore int `json:"seen_before"`
}

type freshDropsView struct {
	Period    string      `json:"period"`
	Locale    string      `json:"locale"`
	Pulled    int         `json:"pulled"`
	New       int         `json:"new"`
	Drops     []freshDrop `json:"drops"`
	ShowsSeen bool        `json:"includes_seen"`
}

func newNovelFreshDropsCmd(flags *rootFlags) *cobra.Command {
	var o dropsOpts
	var all bool
	cmd := &cobra.Command{
		Use:   "fresh-drops",
		Short: "Show only the top price drops you have not already seen in earlier pulls",
		Long: strings.Trim(`
Use this command for a daily scan that hides drops you already saw in earlier
pulls. It fetches the top-drops feed, stores every drop locally keyed by
product and price, and prints only drops that no earlier pull recorded at
that price. A product that drops again to a new price shows again. Use --all
to print everything with a seen_before count. Do NOT use it for the full
current list; use 'drops' instead.`, "\n"),
		Example: strings.Trim(`
  camel-cli fresh-drops --period daily --min-pct 30
  camel-cli fresh-drops --locale au --category electronics --agent`, "\n"),
		Annotations: map[string]string{"pp:data-source": "live", "mcp:local-write": "true", "pp:typed-exit-codes": "0,2,5,7"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "fresh-drops")
			}
			view := freshDropsView{Period: o.period, Locale: o.locale, Drops: []freshDrop{}, ShowsSeen: all}
			if cliutil.IsVerifyEnv() {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			drops, loc, err := fetchDrops(ctx, flags, o)
			if err != nil {
				return err
			}
			view.Locale = loc
			st, db, err := openCamelDB(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			seen, err := camel.RecordDrops(ctx, db, loc, o.period, drops, time.Now())
			if err != nil {
				return err
			}
			view.Pulled = len(drops)
			seenOut := []int{}
			for i, d := range drops {
				if d.ASIN == "" || d.Price == nil {
					// Unparsed feed items cannot be tracked; leave them out.
					continue
				}
				if seen[i] == 0 {
					view.New++
				}
				if seen[i] > 0 && !all {
					continue
				}
				view.Drops = append(view.Drops, freshDrop{Drop: d, SeenBefore: seen[i]})
				seenOut = append(seenOut, seen[i])
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%d new of %d drops (%s, %s)\n\n", view.New, view.Pulled, view.Period, view.Locale)
			plain := make([]camel.Drop, len(view.Drops))
			for i, d := range view.Drops {
				plain[i] = d.Drop
			}
			if len(plain) == 0 {
				fmt.Fprintln(w, "Nothing new since your last pull.")
				return nil
			}
			return renderDrops(cmd, view.Locale, plain, seenOut)
		},
	}
	addDropsFlags(cmd, &o)
	cmd.Flags().BoolVar(&all, "all", false, "Print every drop with a seen_before count, not only new ones")
	return cmd
}
