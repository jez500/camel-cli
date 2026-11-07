# Phase 4.85 output review findings

status: WARN (2 warnings). Live samples: compare, trend, lows passed; fresh-drops skipped (writes local data).

1. ranking-ordering (warning): verdict "good" fired for any price below average (B07T32R43P used: 0.55% below average, 121.9% above lowest -> "good").
   Fixed: good now requires current <= average*0.95; the band from 5% below to 10% above average is "fair". Test case added (internal/camel/camel_test.go TestAssess). Docs and deal --help updated.
2. format-bugs (warning): `lows` printed a bare [] with an empty watchlist in machine mode with no hint.
   Fixed: stderr hint naming 'watch add' then 'watch check' (stdout stays valid JSON). Human mode already printed guidance.
