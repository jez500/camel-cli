# Phase 4.95 local code review

- Review path: direct subagent dispatch (general-purpose agents with correctness, security, maintainability personas). Rounds 1-2: three reviewers in parallel. Round 3: one combined reviewer covering all three lenses on the small round-3 delta (deviation from "all reviewers every round", chosen because the delta was ~10 small edits).
- Autofix summary: 41 findings autofixed in place across 2 fix rounds (round 1: 24 = 7 correctness, 5 security, 12 maintainability minus 2 deferred; round 2: 15 = 1 security, 4 correctness, 8 maintainability, 2 overlap). No git repo in the working dir; the diffs are the record.
- Convergence: findings cleared at round 3 ("No findings").
- Deferred (not fixed, by decision):
  - maintainability: split ExtractSeries/readLine into smaller functions (pure refactor, no behavior change; extraction is covered by fixture + live ground-truth tests).
  - security (informational): `chart` over MCP writes <ASIN>-<locale>-<range>.png into the server working directory by default; filename components are validated (no traversal), `--output` is blocked over MCP, and the tool carries no read-only hint so hosts prompt.
- Post-fix /simplify: skipped; maintainability rounds already consolidated duplicates and round 3 was clean.
- Template-shape retro candidates:
  - generator template pins golang.org/x/net v0.60.0 (GO-2026-6860); every `generate --validate` fails govulncheck and leaves a .preserve snapshot.
  - generated `promoted_chart.go` binary endpoint prints a base64 JSON envelope on stdout by default (PNG only via --deliver file:); consider raw bytes when stdout is not a terminal.
  - generated framework creates an empty data.db and learn journal on every invocation, including ones that exit 2 on flag validation.
- Out-of-scope retro candidates: none.
- Surface-to-user findings: none.
