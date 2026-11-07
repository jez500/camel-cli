# Polish (Phase 5.5)

Polish pass:
  Verify:      100% -> 100% (38/38, 0 critical)
  Scorecard:   81 -> 82 (MCP Desc Quality 5 -> 10)
  Tools-audit: 2 -> 0 pending findings
  gosec (hand-written): 7 -> 0
Fixed: gosec G104 x6 (explicit discards with reasons), G306 chart PNG 0644 (#nosec with reason), watch list Short, mcp-descriptions.json for popular_list/drops_list. Polish reverted mcp-sync and dogfood doc re-sync side effects that removed hand edits.
Skipped: dead generated helpers (requiredInputWaived, retainCLIQueryParams); 45 gosec findings in generated files (retro); Cache Freshness 3/10 by design (no sync resource).
ship_recommendation: ship
Post-polish: phase5 live acceptance re-minted: 139/139 PASS.
