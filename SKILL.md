---
name: pp-camelcamelcamel
description: "Paste an Amazon URL (US, AU, UK, CA, DE, FR, IT or ES) and get its price history: exact lowest, highest, current and average for every price type, plus an approximate dated series read from the chart. Trigger phrases: `price history for this amazon link`, `is this a good price on amazon`, `camelcamelcamel this`, `lowest price ever for this product`, `amazon price drops today`, `use camelcamelcamel`."
author: "Jeremy Graham"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - camel-cli
    install:
      - kind: go
        bins: [camel-cli]
        module: github.com/jez500/camel-cli/cmd/camel-cli
---

# camelcamelcamel — camel-cli

## Prerequisites: Install the CLI

This skill drives the `camel-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Download the archive for your platform from https://github.com/jez500/camel-cli/releases/latest and put `camel-cli` on `$PATH`.
2. Verify: `camel-cli --version`

Or install with Go (requires Go 1.26.10 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH`:

```bash
go install github.com/jez500/camel-cli/cmd/camel-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

It reads camelcamelcamel's public price pages for eight Amazon marketplaces. history gives the exact lifetime summary and a dated price series reconstructed from the chart image (approximate, about a day per pixel). deal gives a buy/wait verdict, compare puts marketplaces side by side, and every lookup saves a snapshot that trend and lows analyze offline.

## When to Use This CLI

Use camel-cli when an agent or user asks whether an Amazon price is good, what the lowest or highest price was, or how a product's price compares across Amazon marketplaces. It also covers daily deal scans (drops, popular, fresh-drops) and a local watchlist.

## Anti-triggers

Do not use this CLI for:
- Searching Amazon by keyword (site search is blocked; get the product URL or ASIN first)
- Buying, carting or checking out on Amazon
- Exact day-by-day prices: the dated series is approximate (read from the chart image, about one day per pixel); only the lowest, highest, current and average values are exact
- Marketplaces other than us, au, uk, ca, de, fr, it, es (for example amazon.co.jp)
- Walmart or other non-Amazon stores

## Unique Capabilities

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

`--ca-cert`, `CAMELCAMELCAMEL_CA_CERT`, or `ca_cert` adds a PEM CA bundle to the system trust roots. `--tls-server-name`, `CAMELCAMELCAMEL_TLS_SERVER_NAME`, or `tls_server_name` is the name checked against the certificate. Precedence is flag, then environment, then the config file. A config file other users can change does not supply these settings. `doctor` reports the trust in use and names the setting that fixes a chain or hostname failure.

## Command Reference

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

### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
camel-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query. `--json` (and other machine formats) keep that exit-2 contract and write `{"matches":[]}` on stdout so agents can inspect the envelope without treating a miss as success.

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

## Auth Setup

No account or API key is needed. camel-cli sends an honest camel-cli User-Agent and about one request per second (history fetches the page plus one chart image; compare fetches one page per marketplace). Site search is not available because Cloudflare challenges every non-browser client on that page.

Run `camel-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color`.

Global format flags share one contract on promoted, novel, and `--deliver` paths:

- `--json` — one JSON document on stdout
- `--compact` — keep identity/status/timestamp fields; does not change the document vs stream shape
- `--csv` / `--plain` — tabular rows (collection envelopes unwrap to the row array). `--csv` prefixes `'` on a cell or header key that starts with `=`, `+`, `-`, `@`, tab, CR, or LF (fullwidth `＝＋－＠` too) so a spreadsheet does not run it as a formula; a plain signed number such as `-5` or `+2.5e3` is unchanged.
- `--quiet` — one identity value per row, no envelope

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  camel-cli history B07T32R43P --no-series --agent --select product.title,verdicts.label,verdicts.verdict
  ```
- **Previewable** — `--dry-run` shows the request without sending. A completed dry run exits 0 with `"success": true` and `"dry_run": true` and no HTTP `status`; validation failures still exit non-zero with `"success": false`.
- **Offline-friendly** — `trend` and `lows` read saved snapshots from the local SQLite store with no network call
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Output shape

The camelcamelcamel commands (`history`, `deal`, `compare`, `chart`, `trend`, `lows`, `watch`, `drops`, `popular`, `fresh-drops`) print their result object or array directly on stdout under `--json` / `--agent`; there is no `{meta, results}` wrapper. The generated endpoint commands (`product`, `drops list`, `drops feed`, `popular list`, `popular feed`) wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

A human-readable summary is printed to stderr only when stdout is a terminal and no machine-format flag is set.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `CAMELCAMELCAMEL_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `CAMELCAMELCAMEL_CONFIG_DIR`, `CAMELCAMELCAMEL_DATA_DIR`, `CAMELCAMELCAMEL_STATE_DIR`, `CAMELCAMELCAMEL_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `CAMELCAMELCAMEL_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains durable local data such as `data.db`. `state` contains runtime state such as the learn journal and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Run `camel-cli doctor --fail-on warn` to surface path warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `CAMELCAMELCAMEL_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `CAMELCAMELCAMEL_HOME`, or `doctor` will not find files left under the former root.

Local data is unencrypted and relies on OS account permissions plus disk encryption. Reads in the default `--data-source auto` mode are written to the SQLite store; GET responses are cached for five minutes under the cache dir. `local-data purge` previews what it would erase (store files, response cache, learn journal) and deletes only with `--yes`; it is not exposed over MCP. `CAMELCAMELCAMEL_NO_LOCAL_DATA=1` disables the response cache and store writes, turns `auto` reads into live reads; `--data-source local` still reads an existing store, and `CAMELCAMELCAMEL_DATA_SOURCE` sets the `--data-source` default.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, pass the question as an argv or MCP tool argument to `recall --agent`. Do not interpolate user-controlled text into a shell command line.

Quoted `recall "<question>"` breaks on an apostrophe, which is ordinary English. A quoted heredoc breaks when a body line equals the delimiter, and that delimiter is published in these docs. Write the question with a non-shell file-writing tool, then read it back as data:

```bash
# Write the question verbatim with your file-writing tool (no shell involved).
# Command substitution on a file only ever yields data — the shell never
# parses the file's bytes as syntax.
QUERY=$(cat /path/to/question.txt)
camel-cli recall "$QUERY" --agent
```

Prefer MCP: pass the question as the tool's query argument. `"$QUERY"` after a file read is argv-safe; putting the question itself in the command text is not.

The response envelope:

```json
{
  "query": "...",
  "normalized": "<normalized form>",
  "query_entities": ["..."],
  "found": true | false,
  "match_score": 0.0,
  "results": [
    { "resource_id": "...", "resource_type": "...", "venue": "...",
      "confidence": 2, "entity_match": "exact|partial|unknown",
      "source": "taught|preseed|pattern", "warnings": ["..."] }
  ],
  "mismatches": [ /* only when --debug-mismatches */ ],
  "warnings": [ /* top-level */ ],
  "candidates": [
    { "id": 12, "class": "flag_alias | playbook_candidate",
      "summary": "...", "sightings": 3, "last_seen": "...",
      "rationale": "...",
      "next_action": ["<trial command>", "camel-cli learnings confirm 12"] }
  ],
  "playbook": {
    "query_family": "...",
    "playbook": {
      "steps": [ { "cmd": "<command with {slot} substitution>", "purpose": "..." } ],
      "entity_slots": ["$ENTITY"],
      "expected_tool_calls": 3
    },
    "slots_resolved": { "$ENTITY": { "token": "<live token>", "canonical": "<canonical>" } },
    "notes": "<workarounds + gotchas for this query family>"
  },
  "notes": "<duplicate surface for non-playbook callers>"
}
```

Empty-store short-circuit: if the store has no learnings, playbooks, or candidates yet (recall finds nothing and `learnings list` and `learnings candidates` are both empty), skip recall for the rest of this session instead of taxing every query; resume recall-first once something has been taught.

### Step 2: decision tree

Read `candidates`, `playbook`, `notes`, `results[0]`, and warnings in that order:

```
if Candidates present (warnings include "candidates_present"):
    -> candidates are try-then-confirm, never facts. Follow each candidate's
       two-step next_action verbatim: run the trial command first, then run
       `learnings confirm <id>` only after the trial verified the behavior.
       Reject a wrong candidate with `learnings reject <id>`.
    -> NEVER re-teach something recall surfaced as a candidate; confirm or
       reject that candidate instead of teaching a duplicate.
    -> candidates ride alongside playbooks and resource hits, not instead of
       them; continue with the branches below after acting on them.

if Playbook present:
    -> READ Playbook.notes verbatim FIRST (workarounds + gotchas the CLI surface doesn't expose)
    -> replay Playbook.steps in order, substituting Playbook.slots_resolved entries
       for the entity slot tokens. If a step's slot is unresolved, fall back to
       discovery for that step only.
    -> the Playbook's expected_tool_calls is a budget; if you find yourself running
       materially more, record the divergence via `camel-cli playbook amend`
       at end-of-session.

elif Notes present (no Playbook):
    -> read Notes verbatim before any discovery step; they carry known gotchas
       for this query family even when no structured choreography exists yet.

elif Found AND Results[0].EntityMatch == "exact" AND Results[0].Confidence >= 2:
    -> skip discovery; fetch live data for Results[*].ResourceID in parallel

elif Found AND Results[0].EntityMatch == "partial":
    -> candidate hint, NOT a hit; read the resource title to validate before trusting

elif (any row in Mismatches[] when --debug-mismatches was passed):
    -> treat as cold start; the stored learning is for a different entity
       (different canonical resolved from query_entities)

else:  // Found == false, no playbook, no notes
    -> cold start; run discovery normally; teach the answer afterward (Step 4).
       If the family has no playbook yet, that teach auto-synthesizes a
       playbook candidate from this session's journal - you do not need to
       record one by hand.
```

Playbook and Notes are orthogonal to the per-resource path. A recall response can carry both a Playbook AND a `Results[]` hit - use both: the Playbook tells you which choreography to run; the resource hits short-circuit specific steps. Default to skipping `mismatches`; pass `--debug-mismatches` only when investigating cold-start surprises.

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `camel-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately. Pass the query the same way as recall — argv/MCP, or file-then-`$QUERY`. Do not splice the question into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
camel-cli teach --query "$QUERY" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
QUERY=$(cat /path/to/question.txt)
camel-cli teach \
  --query "$QUERY" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
QUERY=$(cat /path/to/question.txt)
camel-cli teach-playbook \
  --query "$QUERY" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`. Pass the query and note as argv/MCP arguments, or write each with a non-shell file tool and read them back (`QUERY=$(cat ...)`, `NOTE=$(cat ...)`). Do not interpolate either string into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
NOTE=$(cat /path/to/note.txt)
camel-cli playbook amend \
  --query "$QUERY" \
  --add-note "$NOTE"
# (append shell `&` to background it)
```

What counts as worth amending: a behavior you OBSERVED this session that future-you would benefit from knowing. Examples worth amending:

- A workaround for a CLI surface that silently drops or misorders a flag.
- An undocumented endpoint shape (response wrapped in `{meta, results}`, payload nested two levels deeper than the docs claim).
- Observed schema drift (a field renamed, an index that shifted between seasons, a category label that the API now returns lower-cased).

What does NOT belong in notes:

- The year-specific or entity-specific answer to the user's question. That's the response, not a learning.
- Per-team / per-athlete / per-row data the playbook already retrieves at runtime.
- Statements that paraphrase what the existing notes already say.

The amend command appends to the family's existing notes with a timestamped marker (`[amend YYYY-MM-DDTHH:MMZ]: <text>`). Multiple amends accumulate; the audit trail is visible. If no playbook exists yet for the family, amend creates a notes-only one (so cold-start corrections still land).

#### PII discipline for amend notes

`playbook amend` notes are designed to potentially flow upstream as shared knowledge in future versions of the Printing Press. Keep them clean of user-identifying content so the upstream-contribution path stays open without retroactive scrubbing:

- **Do NOT embed** paths to user filesystems, personal API keys or tokens, user email addresses, user GitHub handles, or specific query histories tied to a single user.
- **Acceptable**: endpoint shapes, undocumented field names, API gotchas, observed schema drift, workarounds for CLI surfaces, generalizable pagination or retry tactics.

If a correction is only meaningful with user-specific context, it belongs in a personal note, not in the playbook amend.

### Measuring the loop

`camel-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `CAMELCAMELCAMEL_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
camel-cli feedback "the --since flag is inclusive but docs say exclusive"
camel-cli feedback --stdin < notes.txt
camel-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `CAMELCAMELCAMEL_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `CAMELCAMELCAMEL_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. The common sinks are below (`spacefast:<name>` also exists; see `--help`):

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` as a private 0600 file (exclusive randomly named temp in the destination directory, then rename; a planted symlink is not followed). Binary-response commands write decoded payload bytes (not the base64 JSON envelope) and print a small JSON receipt on stdout; `--json`/`--csv` do not refuse when this sink is set. |
| `webhook:<url>` | POST the output body to the URL (`application/json`). Redirects are not followed; a 3xx response fails the delivery and the body is not sent to the `Location`. |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures, including redirects, return non-zero and log the URL and HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled or recurring agent reuses the same saved flags while providing different input each run.

```
camel-cli profile save briefing --json
camel-cli --profile briefing history B07T32R43P --no-series
camel-cli profile list --json
camel-cli profile show briefing
camel-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited or blocked by a firewall or bot challenge (back off; do not retry immediately) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `camel-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/jez500/camel-cli/cmd/camel-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add camel-mcp -- camel-mcp
   ```
3. Verify: `claude mcp list`

For a remote host, `camel-mcp --transport http` requires `CAMELCAMELCAMEL_MCP_HTTP_TOKEN` set to at least 32 characters (callers send `Authorization: Bearer <token>`; `openssl rand -hex 32` prints 64 hex characters and meets that minimum), binds `127.0.0.1:7777` by default, and needs `--tls-cert`/`--tls-key` for any non-loopback `--addr`. Requests with no `Origin` header are accepted. Loopback origins are accepted; any other `Origin` is refused unless listed in `CAMELCAMELCAMEL_MCP_HTTP_ALLOWED_ORIGINS` (comma-separated `scheme://host[:port]` values). `CAMELCAMELCAMEL_MCP_READ_ONLY=1` drops mirrored write commands (local-write commands that only touch the CLI's own learn store stay) and makes the raw execute tool refuse state-changing calls.

## Direct Use

1. Check if installed: `which camel-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   camel-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `camel-cli <command> --help`.
