// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jez500/camel-cli/internal/camel"
	"github.com/jez500/camel-cli/internal/cliutil"
	"github.com/jez500/camel-cli/internal/store"
)

// camelDBName is the local store name shared with the generated commands.
const camelDBName = "camel-cli"

// addLocaleFlag registers --locale on a command.
func addLocaleFlag(cmd *cobra.Command, dst *string) {
	cmd.Flags().StringVar(dst, "locale", "us", "Amazon marketplace for bare ASINs: "+strings.Join(camel.LocaleCodes(), ", ")+" (a URL's own marketplace wins unless --locale is set)")
}

func newCamelClient(flags *rootFlags) *camel.Client {
	return camel.NewClient(flags.timeout)
}

// resolveTarget parses an ASIN or URL. An explicit --locale overrides the
// marketplace named by the URL; a short link is resolved over HTTP.
func resolveTarget(ctx context.Context, cmd *cobra.Command, c *camel.Client, input, locale string) (camel.Target, error) {
	if err := checkLocaleFlag(cmd, locale); err != nil {
		return camel.Target{}, err
	}
	t, err := camel.ParseInput(input, locale)
	if err != nil {
		return camel.Target{}, usageErr(err)
	}
	if t.ShortLink != "" {
		t, err = c.ResolveShortLink(ctx, t)
		if err != nil {
			if errors.Is(err, camel.ErrNoASIN) {
				return camel.Target{}, usageErr(err)
			}
			return camel.Target{}, apiErr(err)
		}
	}
	return applyLocaleOverride(cmd, t, locale), nil
}

// resolveLocalTarget parses input for commands that read only the local
// store: no network, so short links are rejected.
func resolveLocalTarget(cmd *cobra.Command, input, locale string) (camel.Target, error) {
	if err := checkLocaleFlag(cmd, locale); err != nil {
		return camel.Target{}, err
	}
	t, err := camel.ParseInput(input, locale)
	if err != nil {
		return camel.Target{}, usageErr(err)
	}
	if t.ShortLink != "" {
		return camel.Target{}, usageErr(fmt.Errorf("%s reads only the local store; pass the full Amazon URL or the ASIN, not a short link", cmd.Name()))
	}
	return applyLocaleOverride(cmd, t, locale), nil
}

// checkLocaleFlag validates an explicitly set --locale before any work.
func checkLocaleFlag(cmd *cobra.Command, locale string) error {
	if !cmd.Flags().Changed("locale") {
		return nil
	}
	if _, err := camel.LookupLocale(locale); err != nil {
		return usageErr(err)
	}
	return nil
}

// applyLocaleOverride makes an explicit --locale win over the URL's marketplace.
func applyLocaleOverride(cmd *cobra.Command, t camel.Target, locale string) camel.Target {
	if cmd.Flags().Changed("locale") {
		if l, err := camel.LookupLocale(locale); err == nil {
			t.Locale = l.Code
		}
	}
	return t
}

// parseSingleType validates a --type flag that takes one price type ("" = all).
func parseSingleType(flag string) (string, error) {
	if flag == "" {
		return "", nil
	}
	t := strings.ToLower(strings.TrimSpace(flag))
	if camel.ValidType(t) {
		return t, nil
	}
	return "", usageErr(fmt.Errorf("invalid --type %q: use one of %s", flag, strings.Join(camel.PriceTypes, ", ")))
}

// validateRange checks a --range value.
func validateRange(rng string) error {
	if !camel.ValidRange(rng) {
		return usageErr(fmt.Errorf("invalid --range %q: use one of %s", rng, strings.Join(camel.Ranges, ", ")))
	}
	return nil
}

// assessAll returns a verdict for every price type with data (never nil).
func assessAll(p *camel.Product) []camel.Deal {
	out := []camel.Deal{}
	for _, ps := range p.Prices {
		if ps.HasData() {
			out = append(out, camel.Assess(ps))
		}
	}
	return out
}

// isRateLimit reports whether err is (or wraps) a rate-limit error.
func isRateLimit(err error) bool {
	var rl *cliutil.RateLimitError
	return errors.As(err, &rl)
}

// fetchProduct fetches one product with its own --timeout deadline, so
// loops over many products are bounded per request, not as a whole.
func fetchProduct(parent context.Context, flags *rootFlags, c *camel.Client, t camel.Target) (*camel.Product, error) {
	ctx, cancel := boundCtx(parent, flags)
	defer cancel()
	return c.FetchProduct(ctx, t)
}

// camelErr maps domain errors to typed exit codes.
func camelErr(err error) error {
	if err == nil {
		return nil
	}
	var ce *cliError
	if errors.As(err, &ce) {
		return err
	}
	switch {
	case errors.Is(err, camel.ErrNotTracked):
		return notFoundErr(err)
	case errors.Is(err, camel.ErrNoASIN):
		return usageErr(err)
	case isRateLimit(err):
		return rateLimitErr(err)
	}
	return apiErr(err)
}

// openCamelDB opens the local store and makes sure the camel tables exist.
func openCamelDB(ctx context.Context) (*store.Store, *sql.DB, error) {
	path := defaultDBPath(camelDBName)
	st, err := store.OpenWithContext(ctx, path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening local store %s: %w", path, err)
	}
	if err := camel.EnsureSchema(ctx, st.DB()); err != nil {
		_ = st.Close() // already returning the schema error
		return nil, nil, err
	}
	return st, st.DB(), nil
}

// openCamelDBReadOnly opens the store without schema changes. ok is false
// when the store or the camel tables do not exist yet (an empty result).
func openCamelDBReadOnly(ctx context.Context) (st *store.Store, db *sql.DB, ok bool, err error) {
	if !camelDBExists() {
		return nil, nil, false, nil
	}
	st, err = store.OpenReadOnlyContext(ctx, defaultDBPath(camelDBName))
	if err != nil {
		return nil, nil, false, fmt.Errorf("opening local store: %w", err)
	}
	var n int
	if err := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('camel_snapshots','camel_watchlist')`).Scan(&n); err != nil || n < 2 {
		_ = st.Close() // tables missing: report an empty result, not a close error
		return nil, nil, false, nil
	}
	return st, st.DB(), true, nil
}

// camelDBExists reports whether the local store file exists yet.
func camelDBExists() bool {
	_, err := os.Stat(defaultDBPath(camelDBName))
	return err == nil
}

// saveSnapshot stores a lookup; failures only warn, because the lookup itself worked.
func saveSnapshot(ctx context.Context, w io.Writer, p *camel.Product) {
	if cliutil.IsMCPReadOnlyEnv() {
		return
	}
	st, db, err := openCamelDB(ctx)
	if err != nil {
		fmt.Fprintf(w, "warning: snapshot not saved: %v\n", err)
		return
	}
	defer st.Close()
	if err := camel.SaveSnapshot(ctx, db, p, time.Now()); err != nil {
		fmt.Fprintf(w, "warning: snapshot not saved: %v\n", err)
	}
}

var currencySymbols = map[string]string{"USD": "$", "AUD": "$", "CAD": "$", "GBP": "£", "EUR": "€"}

// money formats a price for a locale; nil prints "-".
func money(p *float64, locale string) string {
	if p == nil {
		return "-"
	}
	sym := "$"
	if l, err := camel.LookupLocale(locale); err == nil {
		if s, ok := currencySymbols[l.Currency]; ok {
			sym = s
		}
	}
	return sym + groupThousands(fmt.Sprintf("%.2f", *p))
}

// groupThousands inserts commas: "3600.99" -> "3,600.99".
func groupThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}

func moneyV(v float64, locale string) string { return money(&v, locale) }

func withDate(p *float64, date, locale string) string {
	if p == nil {
		return "-"
	}
	if date == "" {
		return money(p, locale)
	}
	return fmt.Sprintf("%s (%s)", money(p, locale), date)
}

func pct(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%+.1f%%", *p)
}

// staleHint warns on stderr when newest is older than --max-age.
func staleHint(cmd *cobra.Command, flags *rootFlags, newest time.Time, refresh string) {
	if flags.maxAge <= 0 || newest.IsZero() {
		return
	}
	if age := time.Since(newest); age > flags.maxAge {
		fmt.Fprintf(cmd.ErrOrStderr(), "hint: newest snapshot is %s old; run '%s' to refresh (--max-age 0 hides this)\n", age.Round(time.Minute), refresh)
	}
}

// splitCSV splits a comma-separated flag value.
func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// typesWithData returns the price types on the page that have any history.
func typesWithData(p *camel.Product) []string {
	out := []string{}
	for _, ps := range p.Prices {
		if ps.HasData() {
			out = append(out, ps.Type)
		}
	}
	return out
}

// pickSeriesTypes resolves --type for series output: "" means the first type
// with data (Amazon when present), "all" means every type with data.
func pickSeriesTypes(p *camel.Product, flag string) ([]string, error) {
	have := typesWithData(p)
	if len(have) == 0 {
		return []string{}, nil
	}
	switch strings.ToLower(strings.TrimSpace(flag)) {
	case "":
		return have[:1], nil
	case "all":
		return have, nil
	}
	want, err := camel.NormalizeTypes(splitCSV(flag))
	if err != nil {
		return nil, usageErr(err)
	}
	return want, nil
}
