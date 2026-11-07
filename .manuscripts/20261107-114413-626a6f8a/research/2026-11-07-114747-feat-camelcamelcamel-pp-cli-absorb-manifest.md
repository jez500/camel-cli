# camelcamelcamel absorb manifest

Binary: camel-cli. Sources searched: GitHub, npm, PyPI, MCP registries (Glama, LobeHub), Apify, Greasy Fork, ClawHub/claudskills. No camelcamelcamel CLI, SDK, or GitHub MCP exists.

## Absorbed (match or beat everything that exists)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Accept ASIN or Amazon URL (/dp/, /gp/product/, /gp/aw/d/, any marketplace) | Apify radeance/camelcamelcamel-scraper; Glama Amazon Price Tracker MCP | camel-cli history | Offline URL parsing, marketplace auto-detect |
| 2 | Resolve short links (a.co, amzn.to, amzn.eu) | Glama Amazon Price Tracker MCP | (behavior in camel-cli history) follows the redirect to read the ASIN | No browser |
| 3 | Price summary: current, lowest, highest, average with dates | Glama MCP; JanNafta/amazon-mcp get_price_history | camel-cli history | Formatted table, --json |
| 4 | Three price types | camelcamelcamel site | (behavior in camel-cli history) --type amazon,new,used | Filter per type |
| 5 | Product details (title, UPC, EAN, model, category, list price, image) | camelcamelcamel site; Apify | (behavior in camel-cli history) details block | Parsed from meta tags |
| 6 | Price-history chart PNG | Camelizer; Greasy Fork "Amazon Camel Graph"; Apify | camel-cli chart | All ranges and price types, saved to file |
| 7 | Top price drops with filters | camelcamelcamel site; Apify | camel-cli drops | Structured rows |
| 8 | Popular products (deal flag) | camelcamelcamel site; Apify | camel-cli popular | Structured rows, --min-below-avg filter |
| 9 | Multiple marketplaces | Apify | (behavior in camel-cli history) --locale, auto from URL domain | One binary for 8 sites |
| 10 | Buy/wait analysis | JanNafta/amazon-mcp | camel-cli deal | Verdict per price type |
| 11 | Price alerts | camelcamelcamel-alerts skill (account RSS + Telegram) | camel-cli watch | Local watchlist, no login; add/list/remove/check |
| 13 | Dated price series (approximate) reconstructed from the chart PNG | none (no tool has it; Keepa needs a paid key) | (behavior in camel-cli history) --series, --range 1m|3m|6m|1y|all, per price type; calibrated on exact lowest/highest; marked approximate | User requirement: daily data points |
| 12 | Untracked-product detection | camelcamelcamel site (redirect) | (behavior in camel-cli history) typed "not tracked" error | Agents get a clear exit code |

## Transcendence (only possible with our approach)
| # | Feature | Command | Score | Buildability | Why Only We Can Do This | Long Description |
|---|---------|---------|-------|--------------|------------------------|------------------|
| 1 | Cross-marketplace compare | compare <url> --locales us,au | 9/10 | hand-code | No tool aligns one ASIN across regional sites | Use this command to see one product's prices on several Amazon marketplaces side by side. Do NOT use it for a single marketplace; use 'history' instead. |
| 2 | Personal price trend | trend <url> --type amazon | 8/10 | hand-code | No surface exposes data points; local snapshots build your own timeline | Use this command to see how one product's price changed across your own past lookups, read from the local store with no network call. Do NOT use it for the current camelcamelcamel summary; use 'history' instead. |
| 3 | Watchlist near-lows ranking | lows --within 10 | 8/10 | hand-code | Ranks a basket by distance from each item's own low, offline | Use this command to rank all watched products by distance from their lowest-ever price, using stored snapshots with no network call. Do NOT use it to refresh prices or fire alerts; use 'watch check' instead. Do NOT use it for one product's verdict; use 'deal' instead. |
| 4 | Unseen drops | fresh-drops --period daily --min-drop 30 | 7/10 | hand-code | Feed has no read state; local table diffs pulls | Use this command for a daily scan that hides drops you already saw in earlier pulls. Do NOT use it for the full current list; use 'drops' instead. |

## Out of scope
- Site search (Cloudflare blocks every non-browser client; needs live page context).
- camelcamelcamel account price watches and email alerts (login).
- Raw chart data points (no surface exposes them).

## Killed candidates
recurring-drops, cheapest, low-season, triage, bargains (kept as popular flag), drops-watched, share, find, arbitrage (FX).

## User decisions at the gate (2026-11-07)
- User requires dated price data points; otherwise do not build.
- Spike: chart-PNG reconstruction matched the lowest-price date exactly; highest within ~1-6 days on a short spike. User approved building with the reconstructed, approximate series as the core of `history`.
- Calibration method: dashed red (highest) and dashed green (lowest) reference rows give exact y->price; x-axis dark line extent + tracking start date ('view data going back to <date>' on the 'All' toggle) + last-checked date give x->date. Prefer non-reference rows; skip dotted (no data) segments.
