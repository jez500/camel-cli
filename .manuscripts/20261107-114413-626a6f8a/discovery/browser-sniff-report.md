# camelcamelcamel browser-sniff report

## 1. User Goal Flow
- Goal: look up the price history of one Amazon product.
- Steps completed:
  1. Open /product/B07T32R43P (headed Chrome; headless got the Cloudflare challenge).
  2. Click chart range 1m -> chart PNG reload with tp=1m.
  3. Toggle price types New, Used -> chart PNG amazon-new.png, amazon-new-used.png.
  4. Click "show current prices" -> no network call (client-side).
  5. Search "kindle paperwhite" via the search form -> /search?sq=... SSR HTML.
- Secondary flows: top drops (filters t, bn, d), popular (deal=0/1), RSS feeds (direct HTTP).
- Coverage: 5 of 5 planned steps.

## 2. Pages & Interactions
- /product/B07T32R43P: clicked #toggle_tp_summary_1m, price type New, price type Used, "show current prices".
- /search?sq=kindle+paperwhite: entered query in #sq, clicked #sqbtn.
- /top_drops, /popular, /top_drops/feed, /popular.xml: direct HTTP (camel-cli UA).

## 3. Browser-Sniff Configuration
- Backend: browser-use 1.x CLI mode, headed after headless hit the challenge.
- Pacing: 1-2 s between actions. No 429.
- Proxy pattern: not detected.

## 4. Endpoints Discovered
| Method | Path | Status | Content-Type |
|---|---|---|---|
| GET | /product/{asin} | 200 (302 -> / when untracked) | text/html |
| GET | charts.camelcamelcamel.com/{locale}/{asin}/{types}.png?force&zero&w&h&desired&legend&ilt&tp&fo&lang | 200 | image/png |
| GET | /product/{asin}/pwform | 200 | text/html (price-watch form, login) |
| GET | /search?sq= | 200 in browser, 403 challenge for every non-browser client | text/html |
| GET | /top_drops?t&bn&d | 200 | text/html |
| GET | /top_drops/feed?t | 200 | application/xml (RSS) |
| GET | /popular?deal | 200 | text/html |
| GET | /popular.xml?deal | 200 | application/xml (RSS) |

## 5. Traffic Analysis
- Protocols: server-rendered HTML, RSS 2.0, PNG. No XHR/fetch data calls (interceptor captured 0).
- Auth signals: none for public pages. Cookies seen: language, sjc_seg2, locale_redirect, bbtoggle, tz (no cf_clearance issued).
- Parameter evidence: chart `tp` from #toggle_tp_summary_{1m,3m,6m,1y,all}; price types from `?cpf=amazon-new`; drops `t`,`bn`,`d` from the filter form selects; popular `deal` from links.
- Protection: Cloudflare managed challenge keyed on client fingerprint. Go net/http + unique UA (`camel-cli/...`) = 200 for product/chart/lists/feeds. Generic UA, spoofed Chrome UA, Surf Chrome TLS = 403. /search = 403 for all non-browser clients.
- Generation hints: transport MUST be `standard` (not browser-chrome). Search requires live page context -> not shippable.
- Candidate commands: history (product page parse), chart, drops, popular, compare (multi-locale).

## 6. Coverage Analysis
- Covered: product summary, product details, chart, top drops, popular.
- Not covered: search (blocked), price watches/alerts (login, out of scope). No raw chart data points exist on any surface; only summary stats + PNG.

## 7. Response Samples
- Product summary table (text): `Amazon $19.98 (Jun 26, 2026) | $39.99 (Oct 17, 2025) | $19.98 (Nov 06, 2026) | $26.78`, rows for 3rd Party New and 3rd Party Used. Header `Current <span title="Last checked Nov 06, 2026.">`.
- Clean product fields in `<meta name="keywords">`: `Title: ..., ASIN: ..., UPC: ..., EAN: ..., Model: ..., Product group: ..., Category: ..., Locale: US, Browse Node: ...`. og:image holds the product image. Visible "Product Details" text has injected splits ("D VD"); parse meta, not visible text.
- RSS drops item title: `<name> - down 38.10% ($12.30) to $19.98 from $32.28`.
- RSS popular item description: `Current Price: $36.99 - Best Price / List Price: $55.99 / Avg. Price: $46.59`.
- Chart: Binary response: image/png, ~9-11 KB.

## 8. Rate Limiting Events
- None. ~0.5 req/s effective.

## 9. Authentication Context
- No authenticated session used.
