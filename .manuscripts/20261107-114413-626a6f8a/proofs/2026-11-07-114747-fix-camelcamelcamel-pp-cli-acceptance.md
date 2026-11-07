Acceptance Report: camelcamelcamel
  Level: Full Dogfood
  Tests: 139/139 passed (114 skipped by the runner: error paths on live commands, framework commands needing --allow-destructive, missing positional fixtures)
  First run: 136/141 passed, 5 failed
  Failures fixed: 5
    - drops feed, popular feed (happy_path + json_fidelity): spec declared RSS endpoints as response_format binary; the JSON guard rejected the XML body as "HTML". Fixed: response_format xml (generator converts XML to JSON). [CLI fix + Printing Press note: internal-YAML spec-format.md enum omits `xml` although the generator supports it]
    - product error_path: an invalid ASIN returned the camelcamelcamel home page with exit 0 (site redirects unknown products). Fixed: product rejects non-ASIN input with exit 2. [CLI fix]
  Printing Press issues: 3
    - spec-format.md lists response_format json|csv|html|binary; `xml` exists (spec.ResponseFormatXML) but is undocumented.
    - cross-spec `generate --force` (any spec edit) drops hand-edits to generated files and to README/SKILL/AGENTS; needed a reapply script.
    - template pins golang.org/x/net v0.60.0 (GO-2026-6860).
  Coverage of headline/novel commands: history, deal, chart, compare, trend, lows, fresh-drops, drops, popular, watch add/check/list/remove, product: help + live happy_path + json_fidelity all pass.
  Gate: PASS
