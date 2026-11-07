// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
)

type chartView struct {
	ASIN   string   `json:"asin"`
	Locale string   `json:"locale"`
	Types  []string `json:"types"`
	Range  string   `json:"range"`
	URL    string   `json:"chart_url"`
	Output string   `json:"output"`
	Bytes  int      `json:"bytes"`
}

// newChartCmd replaces the generated promoted chart command. It keeps the
// generated three-positional form (<locale> <asin> <price_types>) and adds
// the one-argument form (<amazon-url-or-asin>).
func newChartCmd(flags *rootFlags) *cobra.Command {
	var locale, types, rng, output string
	var width, height int
	var legend bool
	cmd := &cobra.Command{
		Use:   "chart <amazon-url-or-locale> [asin] [price_types]",
		Short: "Save the camelcamelcamel price-history chart (PNG) for a product",
		Long: strings.Trim(`
Downloads the price-history chart image and writes it to a file (default
<ASIN>-<locale>-<range>.png). Use --output - to write the PNG to stdout.

The older form 'chart <locale> <asin> <price_types>' still works.`, "\n"),
		Example: strings.Trim(`
  camel-cli chart https://www.amazon.com/dp/B07T32R43P --range 1y
  camel-cli chart B07T32R43P --types amazon,new --output local-hero.png
  camel-cli chart us B07T32R43P amazon-new-used`, "\n"),
		Annotations: map[string]string{
			"pp:endpoint":         "chart.get",
			"pp:method":           "GET",
			"pp:path":             "/{locale}/{asin}/{price_types}.png",
			"pp:data-source":      "live",
			"pp:happy-args":       "input=B07T32R43P",
			"pp:live-happy-path":  "true",
			"pp:typed-exit-codes": "0,2,3,5,7",
			"mcp:write-flags":     "output",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "chart")
			}
			if err := validateRange(rng); err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c := newCamelClient(flags)
			var t camel.Target
			typeList := splitCSV(types)
			switch len(args) {
			case 1:
				var err error
				t, err = resolveTarget(ctx, cmd, c, args[0], locale)
				if err != nil {
					return err
				}
			case 3:
				l, err := camel.LookupLocale(args[0])
				if err != nil {
					return usageErr(err)
				}
				if !camel.IsASIN(args[1]) {
					return usageErr(fmt.Errorf("invalid ASIN %q", args[1]))
				}
				t = camel.Target{ASIN: strings.ToUpper(args[1]), Locale: l.Code}
				typeList = strings.Split(args[2], "-")
			default:
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass an Amazon URL or ASIN (or <locale> <asin> <price_types>)"))
			}
			norm, err := camel.NormalizeTypes(typeList)
			if err != nil {
				return usageErr(err)
			}
			png, u, err := c.FetchChart(ctx, camel.ChartOptions{Locale: t.Locale, ASIN: t.ASIN, Types: norm, Range: rng, Width: width, Height: height, Legend: legend})
			if err != nil {
				return camelErr(err)
			}
			if output == "-" {
				_, err := cmd.OutOrStdout().Write(png)
				return err
			}
			if output == "" {
				output = fmt.Sprintf("%s-%s-%s.png", t.ASIN, t.Locale, rng)
			}
			if err := os.WriteFile(output, png, 0o644); err != nil { // #nosec G306 -- user-requested public chart image, same mode as curl -o; holds no secrets
				return fmt.Errorf("writing %s: %w", output, err)
			}
			view := chartView{ASIN: t.ASIN, Locale: t.Locale, Types: norm, Range: rng, URL: u, Output: output, Bytes: len(png)}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved %s (%d bytes)\n", output, len(png))
			return nil
		},
	}
	addLocaleFlag(cmd, &locale)
	cmd.Flags().StringVar(&types, "types", "amazon,new,used", "Price types to draw: amazon, new, used (comma list)")
	cmd.Flags().StringVar(&rng, "range", "all", "Time range: 1m, 3m, 6m, 1y, all")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default <ASIN>-<locale>-<range>.png; - for stdout)")
	cmd.Flags().IntVar(&width, "width", camel.DefaultChartWidth, "Image width in pixels (the service renders at 2x)")
	cmd.Flags().IntVar(&height, "height", camel.DefaultChartHeight, "Image height in pixels")
	cmd.Flags().BoolVar(&legend, "legend", true, "Draw the legend")
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		chart := newChartCmd(flags)
		// The generated endpoint command (pp:endpoint chart.get, Use "raw ...")
		// moves under chart as `chart raw` (base64 JSON envelope, or --deliver file:).
		for _, c := range root.Commands() {
			if c.Annotations["pp:endpoint"] == "chart.get" {
				root.RemoveCommand(c)
				chart.AddCommand(c)
				break
			}
		}
		root.AddCommand(chart)
	})
}
