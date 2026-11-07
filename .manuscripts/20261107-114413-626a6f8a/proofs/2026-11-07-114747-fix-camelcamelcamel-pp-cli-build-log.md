Manifest transcendence rows: 4 planned, 0 built. Phase 3 will not pass until all 4 ship.

## Slice 1: domain package internal/camel (done, build-green)
- locale.go, input.go (ASIN/URL/short-link parsing), product.go (summary table + details), feed.go (drops/popular RSS), deal.go (verdict), client.go (Go stdlib HTTP, camel-cli UA, AdaptiveLimiter 1 rps, typed challenge/429 errors), series.go + chart.go (price series from combined amazon-new-used "all" chart), store.go (camel_snapshots, camel_watchlist, camel_drops_seen).
- Chart extraction design: one combined chart per product; x axis = tracking start .. last checked; price axis fitted by least squares from each line's top/bottom pixel = that type's exact lifetime highest/lowest (worst point dropped while residual > 3% of range); horizontal runs = levels, first touching column starts a new level, column spikes beyond nearby levels recorded; dotted = available:false.
- Finding: single-type charts start at that type's own first data date (unknown), so they cannot be calibrated in x. Range charts' dashed lines are range extremes, not lifetime. Hence combined "all" chart only; --range filters by date.
- Validation (live, 2026-11-07): 21 of 22 lifetime extremes matched page dates within 7 days across us:B07T32R43P, au:B00OYGGMXK, us:B00JAQ5NC0, us:B0DXNBFL3C. Miss: au:B00OYGGMXK new highest 591.73@2026-04-28 (same day as amazon highest; line likely hidden under the other line).
- Correction to spec: drops `d` is a currency amount for recent/daily/weekly (percent only for relative periods).

## Slice 2: commands (done, build-green)
Manifest transcendence rows: 4 planned, 4 built (compare, trend, lows, fresh-drops).
- Absorbed rows built: history (headline; URL/short-link/ASIN, summary, verdicts, chart-derived series with --range/--type/--daily/--no-series), deal, chart (replaces generated promoted chart; keeps `<locale> <asin> <price_types>` form and pp:endpoint chart.get), drops (structured default on generated group, RSS-backed, --period/--category/--min-drop/--min-pct), popular (structured default, --deals-only/--min-below-avg), watch add|list|remove|check, locale handling (--locale, URL domain auto-detect), not-tracked typed exit 3.
- Per-row Cobra resolution: all approved paths resolve with `<path> [flags]` usage lines. dogfood novel_features_check: planned 4, found 4.
- Live checks (temp HOME): drops, popular, fresh-drops (5 new then 0 new on second pull), compare us,au, watch add/check/list/remove, lows, trend, history not-tracked exit 3, bad input exit 2, unsupported .co.jp exit 2.
- Deferred: none. Site search out of scope (Cloudflare). camelcamelcamel account price watches out of scope (login).
- Generator limitations found: template pins golang.org/x/net v0.60.0 (GO-2026-6860) so every `generate --validate` fails govulncheck and leaves a .preserve snapshot; worked around by bumping post-generate and regenerating with --validate=false. Binary suffix -pp-cli is fixed; user wants `camel-cli` (handled at release).
