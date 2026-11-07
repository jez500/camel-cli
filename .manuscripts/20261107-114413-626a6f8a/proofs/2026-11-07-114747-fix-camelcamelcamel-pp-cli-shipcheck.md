# camelcamelcamel-pp-cli shipcheck

## Runs
| Run | verify | validate-narrative | dogfood | workflow-verify | verify-skill | scorecard | Verdict |
|---|---|---|---|---|---|---|---|
| 1 | PASS 37/37 (watch exec FAIL row) | PASS | FAIL | PASS | PASS (6 likely false positives) | 80/100 A | FAIL |
| 2 | PASS 38/38 | PASS | PASS | PASS | FAIL (2 errors) | 81/100 A | FAIL |
| 3 | PASS 38/38, 0 critical | PASS (9 narrative commands) | PASS | workflow-pass | PASS | 81/100 A; live sample 3/3 | PASS (7/7) |

## Blockers found and fixes
1. dogfood: unverified novel hosts amzn.asia, amzn.com (short-link hosts) -> documented in discovery/documented-urls.txt; "www" pseudo-host from a "https://www." literal -> AmazonURL now builds a url.URL.
2. dogfood: dead root flag --max-age -> lows and trend print a stderr staleness hint when the newest snapshot is older than --max-age.
3. dogfood: dead generated helpers (were used by the replaced promoted chart) -> generated chart command kept as `chart raw <locale> <asin> <price_types>`; bare-call checks use hasChangedLocalFlags. Two helpers remain unused (requiredInputWaived, retainCLIQueryParams): WARN only.
4. verify: `watch` parent exec probe -> declared pp:typed-exit-codes 0,2.
5. verify-skill: static Use line of the generated chart command -> promoted_chart.go Use renamed to `raw ...`; parent chart Use is `chart <amazon-url-or-locale> [asin] [price_types]`.

## Before/after
- verify pass rate: 37/37 with one exec FAIL row -> 38/38, 0 critical.
- scorecard: 80 -> 81 (Grade A). Weak dims: Cache Freshness 3/10 (no sync resource by design), Dead Code 3/5.

## Behavioral checks (live, temp HOME)
history (US URL, AU URL, --range 1y, --json series), deal, chart (URL form, 3-arg form), compare us,au, trend, lows, watch add/list/remove/check, drops (daily/electronics), popular (--deals-only --min-below-avg 20), fresh-drops (5 new, then 0 new). Not-tracked ASIN -> exit 3; non-ASIN text -> exit 2; amazon.co.jp -> exit 2.
Series accuracy: 21/22 lifetime extremes matched page dates within 7 days across 4 products (us, au).

## Recommendation
ship
