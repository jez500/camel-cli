# camelcamelcamel CLI

**Paste an Amazon URL (US, AU, UK, CA, DE, FR, IT or ES) and get its price history: exact lowest, highest, current and average for every price type, plus an approximate dated series read from the chart.**

It reads camelcamelcamel's public price pages for eight Amazon marketplaces. history gives the exact lifetime summary and a dated price series reconstructed from the chart image (approximate, about a day per pixel). deal gives a buy/wait verdict, compare puts marketplaces side by side, and every lookup saves a snapshot that trend and lows analyze offline.

Learn more at [camelcamelcamel](https://camelcamelcamel.com).

Created by [@jez500](https://github.com/jez500) (Jeremy Graham).

## Install

### Pre-built binary

Download the archive for your platform from the [latest release](https://github.com/jez500/camel-cli/releases/latest) (Linux, macOS and Windows, amd64 and arm64). Each archive holds `camel-cli` and the MCP server `camel-mcp`. On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine camel-cli`.

### With Go

Requires Go 1.26.10 or newer:

```bash
go install github.com/jez500/camel-cli/cmd/camel-cli@latest
go install github.com/jez500/camel-cli/cmd/camel-mcp@latest   # optional MCP server
```

### From source

```bash
git clone https://github.com/jez500/camel-cli && cd camel-cli
make build   # writes bin/camel-cli
```

### Agent skill

`SKILL.md` in this repo is an agent skill (Claude Code and other agents). Copy it into your agent's skills directory, for example `~/.claude/skills/camel-cli/SKILL.md`.

## Releases

GitHub Actions builds the binaries. Push a version tag to publish a release:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

The `release` workflow runs GoReleaser and attaches archives for six platforms plus `checksums.txt` to the GitHub release. The `ci` workflow runs `go vet` and `go test` on every push and pull request.

## Use with Claude Desktop

Install the MCP server (`camel-mcp`, included in each release archive or via `go install`), then add it to your Claude Desktop config.

```bash
go install github.com/jez500/camel-cli/cmd/camel-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "camelcamelcamel": {
      "command": "camel-mcp"
    }
  }
}
```

### Remote MCP over HTTP

`camel-mcp --transport http` (or `PP_MCP_TRANSPORT=http`) serves streamable HTTP at `http://127.0.0.1:7777/mcp`. It refuses to start until `CAMELCAMELCAMEL_MCP_HTTP_TOKEN` is set to at least 32 characters, and every request must send `Authorization: Bearer <token>`. The command below prints 64 hex characters and meets that minimum. Plain HTTP is allowed only on a loopback `--addr`; any other bind, including `:7777` (every interface), also needs `--tls-cert` and `--tls-key`. Requests with no `Origin` header are accepted, as are loopback origins (`localhost`, `127.0.0.1`, or `[::1]`, any port). Any other `Origin` is refused unless it is listed in `CAMELCAMELCAMEL_MCP_HTTP_ALLOWED_ORIGINS`, a comma-separated list of `scheme://host[:port]` values with no path. That list is not a CORS policy; the server does not emit CORS headers.

```bash
export CAMELCAMELCAMEL_MCP_HTTP_TOKEN="$(openssl rand -hex 32)"
camel-mcp --transport http
```

Set `CAMELCAMELCAMEL_MCP_READ_ONLY=1` (exactly `1`) to drop mirrored commands that are not marked read-only and make the raw execute tool refuse state-changing calls. Commands marked local-write, which write only this CLI's own learn store, stay available.

## Authentication

No account or API key is needed. camel-cli sends an honest camel-cli User-Agent and about one request per second (history fetches the page plus one chart image; compare fetches one page per marketplace). Site search is not available because Cloudflare challenges every non-browser client on that page.

## Quick Start

```bash
# Check the install without any network call
camel-cli doctor --dry-run


# The headline: price history for an Amazon URL
camel-cli history https://www.amazon.com/dp/B07T32R43P


# Buy-or-wait verdict from current vs lowest and average
camel-cli deal B07T32R43P


# Save the price-history chart image
camel-cli chart B07T32R43P --range 1y --output chart.png


# Same product on two marketplaces
camel-cli compare B0BSHF7WHW --locales us,au

```

## Unique Features

### Better buying decisions

- **`compare`** — See one product's current, lowest and average prices on the US, AU and other Amazon sites side by side.

  _Pick this when the question is where a product is cheapest relative to its own history._

  ```bash
  camel-cli compare https://www.amazon.com.au/dp/B0BSHF7WHW --locales us,au --agent
  ```

### Offline analysis of saved snapshots

- **`trend`** — See how a product's price moved across your own past lookups (history, deal, compare, watch add and watch check save snapshots), including new all-time lows.

  _Pick this to answer 'has the price changed since I last checked?' without a network call._

  ```bash
  camel-cli trend B07T32R43P --type amazon --agent
  ```
- **`lows`** — Rank every watched product by how close its current price is to its lowest-ever price (needs watch add and watch check first).

  _Pick this to answer 'which of my watched items should I buy now?'_

  ```bash
  camel-cli lows --within 10 --agent
  ```
- **`fresh-drops`** — Show only the top price drops you have not already seen in earlier pulls.

  _Pick this for a daily deal scan that skips items already reviewed._

  ```bash
  camel-cli fresh-drops --period daily --min-pct 30 --agent
  ```

## Recipes


### Price history for a pasted URL

```bash
camel-cli history 'https://www.amazon.com.au/Alpine-PWE-S8/dp/B00OYGGMXK'
```

Detects the AU marketplace from the domain and prints the three price types.

### Just the verdicts for an agent

```bash
camel-cli history B07T32R43P --no-series --agent --select product.title,verdicts.label,verdicts.verdict,verdicts.current
```

Skips the chart and narrows the JSON to the title and one verdict per price type.

### Daily unseen drops in electronics

```bash
camel-cli fresh-drops --period daily --category electronics --min-pct 30 --agent
```

Shows only electronics drops of 30% or more that earlier pulls did not record at that price.

### Which watched items are near their low

```bash
camel-cli lows --within 10
```

Ranks watched products from their stored snapshots; run watch add and watch check first.

## Usage

Run `camel-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as the learn journal and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `CAMELCAMELCAMEL_CONFIG_DIR`, `CAMELCAMELCAMEL_DATA_DIR`, `CAMELCAMELCAMEL_STATE_DIR`, or `CAMELCAMELCAMEL_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `CAMELCAMELCAMEL_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export CAMELCAMELCAMEL_HOME=/srv/camelcamelcamel
camel-cli doctor
```

Under `CAMELCAMELCAMEL_HOME=/srv/camelcamelcamel`, the four dirs resolve to `/srv/camelcamelcamel/config`, `/srv/camelcamelcamel/data`, `/srv/camelcamelcamel/state`, and `/srv/camelcamelcamel/cache`.

On Windows, set the same variable in PowerShell:

```powershell
$env:CAMELCAMELCAMEL_HOME = "C:\srv\camelcamelcamel"
camel-cli doctor
```

Under `$env:CAMELCAMELCAMEL_HOME = "C:\srv\camelcamelcamel"`, the four dirs resolve to `C:\srv\camelcamelcamel\config`, `C:\srv\camelcamelcamel\data`, `C:\srv\camelcamelcamel\state`, and `C:\srv\camelcamelcamel\cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "camelcamelcamel": {
      "command": "camel-mcp",
      "env": {
        "CAMELCAMELCAMEL_HOME": "/srv/camelcamelcamel"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `CAMELCAMELCAMEL_DATA_DIR` overrides an explicit `--home` for that kind. Use `CAMELCAMELCAMEL_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `CAMELCAMELCAMEL_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `camel-cli doctor --fail-on warn` to check path warnings in automation.

### Local data

Reads are saved locally by default: in `--data-source auto` mode every successful read is also written to the SQLite store (`data.db`) in the data directory. GET responses are cached for five minutes under `http/` in the cache directory. The learn journal lives under `learn/` in the state directory. `camel-cli doctor` prints the resolved paths, every store file, and the response-cache size.

None of it is encrypted. New files are created owner-only; beyond that the data relies on OS account permissions and full-disk encryption (FileVault, BitLocker, LUKS).

- **Erase it:** `camel-cli local-data purge` lists the store files, response cache, and learn journal it would delete; add `--yes` to delete them, or `--only store|cache|journal` to narrow it. Config is kept.
- **Run without a local copy:** set `CAMELCAMELCAMEL_NO_LOCAL_DATA=1`. Reads go to the camelcamelcamel website (`--data-source live`; `--data-source local` still reads an existing store), the response cache is off (`--no-cache`), nothing new is written to the store. Set it in an MCP host's `env` block and every command the server runs inherits it.
- **Choose the read source without flags:** `CAMELCAMELCAMEL_DATA_SOURCE=auto|live|local` sets the `--data-source` default; an explicit flag wins.

## Commands

Inputs: every product command takes an Amazon product URL (`/dp/`, `/gp/product/`, `/gp/aw/d/`, or an a.co / amzn.to / amzn.eu short link) or a bare 10-character ASIN. The marketplace comes from the URL domain; for a bare ASIN use `--locale` (us, au, uk, ca, de, fr, it, es; default us). Other marketplaces (for example amazon.co.jp) exit 2.

**Price history (live)**

- `camel-cli history <url-or-asin> [--range 1m|3m|6m|1y|all] [--type amazon,new,used|all] [--daily] [--no-series]` — Exact lifetime summary per price type (lowest and highest with dates, current, average), a buy/wait verdict per type, and a dated price series. The series is approximate: it is read from the camelcamelcamel chart image (about one day per pixel; `available:false` marks stretches with no price). Default series type is Amazon, else the first type with data. Exit 3 when camelcamelcamel does not track the ASIN on that marketplace.
- `camel-cli deal <url-or-asin>` — One request, no chart. Verdict per price type: buy (within 3% of the lowest ever), good (at least 5% below average), fair (5% below to 10% above average), wait (more than 10% above average). The headline verdict uses the cheapest type with a current price.
- `camel-cli compare <url-or-asin> --locales us,au` — The same ASIN on several marketplaces side by side, in each marketplace's own currency (no conversion). One page request per marketplace.
- `camel-cli chart <url-or-asin> [--range 1y] [--types amazon,new] [--output file.png|-]` — Save the chart PNG (default file `<ASIN>-<locale>-<range>.png`). The older form `chart <locale> <asin> <price_types>` still works. `chart raw <locale> <asin> <price_types>` is the generated endpoint form: it prints a base64 JSON envelope, or writes the PNG with `--deliver file:chart.png`.
- `camel-cli product <asin>` — Raw page metadata for the product page (generated endpoint form; use `history` for parsed prices).

**Watchlist and saved snapshots (local store)**

Snapshots are saved by `history`, `deal`, `compare`, `watch add` and `watch check`.

- `camel-cli watch add <url-or-asin>...` — Add products (each is fetched once to confirm camelcamelcamel tracks it). Products that fail are reported per item.
- `camel-cli watch check [--only buy,good]` — Fetch every watched product (about one per second), save a snapshot, print a verdict per product.
- `camel-cli watch list` / `watch remove <url-or-asin>` — Local only.
- `camel-cli lows [--within 10] [--type amazon]` — Rank watched products by distance from their lowest-ever price, from the latest snapshots. Needs `watch add` and `watch check` first; otherwise it returns an empty list.
- `camel-cli trend <url-or-asin> [--type amazon]` — Your own snapshots for one product over time, with the change between snapshots and new all-time lows. Empty until a lookup has saved a snapshot.

**Deals across Amazon (live)**

- `camel-cli drops [--locale us] [--period recent|daily|weekly|relativedaily|relativeweekly] [--category electronics] [--min-pct 30] [--min-drop 10]` — Top price drops with drop percent, price and previous price. `--min-drop` is a site filter: a currency amount (1, 5, 10, 25, 50, 100) for recent/daily/weekly, or a percent for the relative periods. `--min-pct` filters locally. Titles come truncated from the feed.
- `camel-cli fresh-drops [same flags] [--all]` — Only drops that earlier pulls did not record at that price.
- `camel-cli popular [--deals-only] [--min-below-avg 20]` — Popular products with current, list and average price.
- `drops feed`, `drops list`, `popular feed`, `popular list` — Raw RSS (as JSON) / page forms (generated endpoint commands).

Not available: site search (Cloudflare blocks it for non-browser clients) and camelcamelcamel account price alerts (login required).

### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`camel-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`camel-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`camel-cli learnings list`** - Inspect taught rows
- **`camel-cli learnings forget <query>`** - Undo a teach
- **`camel-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`camel-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`camel-cli teach-pattern`** - Install a query/resource template up front
- **`camel-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `CAMELCAMELCAMEL_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `camel-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
camel-cli history B07T32R43P --no-series

# JSON for scripting and agents
camel-cli history B07T32R43P --no-series --json
# Filter to specific fields by name
camel-cli history B07T32R43P --no-series --json --select product.title,verdicts.verdict

# Dry run — show the request without sending
camel-cli history B07T32R43P --no-series --dry-run

# Agent mode — JSON + compact + no prompts in one flag
camel-cli history B07T32R43P --no-series --agent
```

`--csv` prefixes a leading `'` on cells and header keys that start with `=`, `+`, `-`, `@`, a tab, or a line break (including fullwidth `＝＋－＠`) so a spreadsheet does not run them as formulas. Plain signed numbers such as `-5` and `+2.5e3` are left unchanged.

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - `trend` and `lows` read saved snapshots from the local SQLite store with no network call
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited or blocked, `10` config error.

## Health Check

```bash
camel-cli doctor
```

Verifies configuration and connectivity to the camelcamelcamel website.

## Configuration

Run `camel-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/camel-cli/config.toml` on macOS/Linux and `%USERPROFILE%\.config\camel-cli\config.toml` on Windows; `--home`, `CAMELCAMELCAMEL_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

`--ca-cert`, `CAMELCAMELCAMEL_CA_CERT`, or `ca_cert` adds a PEM CA bundle to the system trust roots. `--tls-server-name`, `CAMELCAMELCAMEL_TLS_SERVER_NAME`, or `tls_server_name` is the name checked against the certificate when the dialed address does not match it. Precedence is flag, then environment, then the config file. A config file other users can change does not supply these settings. Certificate verification failures are not retried.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the ASIN or URL and `--locale`; exit 3 from `history`/`deal` means camelcamelcamel does not track that ASIN on that marketplace (try another `--locale`)

### API-specific

- **not tracked by camelcamelcamel** — camelcamelcamel has no history for that ASIN on that marketplace; try another --locale or add it on the website
- **HTTP 403 with a Cloudflare challenge** — Wait a few minutes and retry; do not change the User-Agent to a browser string, because that triggers the challenge
- **wrong marketplace** — Pass --locale au (or us, uk, ca, de, fr, it, es) when giving a bare ASIN

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
