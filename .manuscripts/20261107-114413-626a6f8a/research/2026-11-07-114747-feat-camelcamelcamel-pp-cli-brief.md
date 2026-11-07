# camelcamelcamel CLI Brief

## API Identity
- Domain: Amazon price tracking. camelcamelcamel.com (US) plus regional sites (au, uk, ca, de, fr, it, es, jp). Each site tracks the matching Amazon marketplace.
- Users: shoppers who want to know if an Amazon price is good; deal hunters; agents answering "should I buy this now?".
- Data profile: per-product price summary for three price types (Amazon, 3rd Party New, 3rd Party Used): lowest ever (with date), highest ever (with date), current (with date), average. Product metadata (title, ASIN, UPC, EAN, model, product group, category, list price, image). A PNG price-history chart. Site-wide lists: top drops, popular products.
- Homepage: https://camelcamelcamel.com (public website; no official public API exists).

## Reachability Risk
- Medium. Cloudflare protects the site (`cf-mitigated: challenge`).
- Evidence (probed 2026-11-07 from this machine with Go net/http):
  - A unique, honest User-Agent (`camel-cli/0.1`) gets HTTP 200 on `/product/<ASIN>`, `/top_drops`, `/popular`, `charts.camelcamelcamel.com/<locale>/<ASIN>/<types>.png`, and `/top_drops/feed`. Repeated 3x, stable.
  - Generic UAs (`Go-http-client/1.1`, `curl/*`) and spoofed Chrome UAs get 403 challenge. Surf (Chrome TLS) also gets 403.
  - `/search?sq=` gets 403 challenge with every UA tested. Search is out of the no-browser runtime.
- Robots.txt: `User-agent: * Allow: /`, `Content-Signal: search=yes, ai-train=no, use=reference`. A personal lookup CLI fits "reference" use. Keep request volume low (one page per lookup, polite delay in batch mode, local cache).
- Untracked ASINs: `/product/<ASIN>` redirects (302) to `/`. The CLI must detect this and report "not tracked".

## Top Workflows
1. Paste an Amazon URL (any form: /dp/, /gp/product/, /gp/aw/d/, amazon.com.au, a.co short link) and get a formatted price history: current, lowest, highest, average per price type, with dates.
2. "Is this a good deal now?": compare current price to lowest/average and give a verdict.
3. Download or display the price-history chart (PNG) for a time range (1m/3m/6m/1y/all) and price types.
4. Check the same product across marketplaces (US vs AU).
5. Browse today's top price drops / popular products, optionally filtered.

## Table Stakes
- ASIN extraction from every Amazon URL form, plus raw ASIN input.
- Locale auto-detection from the Amazon domain; `--locale` override.
- Summary table for the three price types with dates.
- Product details (title, group, category, list price, UPC/EAN).
- Chart PNG download (the Camelizer and the site both offer it).
- Top drops and popular lists (site + RSS feed).
- `--json` output for agents.

## Data Layer
- Primary entities: product (asin+locale), price_snapshot (asin, locale, price_type, current/lowest/highest/average + dates, fetched_at), drop (from top_drops feed).
- Sync cursor: fetched_at per product; watchlist of tracked products.
- FTS/search: product titles in the local store (offsets the blocked site search).

## User Vision
- Binary name `camel-cli`. Repo on GitHub. GitHub Actions builds release binaries.
- Headline: give an Amazon URL, get back a formatted price history.
- Sites: camelcamelcamel.com and au.camelcamelcamel.com (other locales are a cheap extra).

## Product Thesis
- Name: camel-cli
- Why it should exist: no CLI or keyless tool exists for camelcamelcamel. Keepa needs a paid key. Existing MCP servers use headless Chromium or are "best-effort". A single Go binary with an honest UA, URL parsing, a deal verdict, and a local snapshot history answers "is this a good price?" in one command.

## Competitors
- Camelizer browser extension (official; in-browser only).
- Keepa (paid API, keepa-mcp).
- JanNafta/amazon-mcp (camelcamelcamel source marked best-effort).
- Glama "Amazon Price Tracker MCP" (Playwright headless browser to pass Cloudflare).
- Apify radeance/camelcamelcamel-scraper (hosted, paid).

## Build Priorities
1. `history <amazon-url|asin>`: parse URL, detect locale, fetch product page, parse table + details, render table / JSON. Not-tracked detection.
2. `deal` verdict (inside history output and as its own command).
3. `chart` download (PNG) with range and price types; optional inline terminal display.
4. `compare` across locales; `drops` / `popular` lists.
5. Local snapshot store + `watch` list for repeat checks.

## Reachability Gate
- Decision: PASS. Go net/http with UA `camel-cli/0.1.0`: /product/{asin} 200, /top_drops/feed 200 (2026-11-07).
- curl with the same UA gets 403 on /product (curl TLS fingerprint is challenged). Surf/Chrome TLS also 403. The printed CLI MUST use Go stdlib transport (`--transport standard`), never browser-chrome/uTLS.

## Input URL Hosts (parsed offline, product URLs the user pastes)
- https://www.amazon.com/dp/B07T32R43P (US), https://www.amazon.com.au/dp/B0BSHF7WHW (AU), https://www.amazon.co.uk, https://www.amazon.ca, https://www.amazon.de, https://www.amazon.fr, https://www.amazon.it, https://www.amazon.es
- Short links: https://a.co, https://amzn.to, https://amzn.eu (one redirect followed to read the ASIN)
- Regional camelcamelcamel hosts: https://au.camelcamelcamel.com, https://uk.camelcamelcamel.com, https://ca.camelcamelcamel.com, https://de.camelcamelcamel.com, https://fr.camelcamelcamel.com, https://it.camelcamelcamel.com, https://es.camelcamelcamel.com, https://charts.camelcamelcamel.com
