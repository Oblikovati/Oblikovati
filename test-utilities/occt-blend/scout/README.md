# OCCT blend-parity scout probe

A throwaway-by-design diagnostic that answers the one question the corpus scoreboard cannot:
**why** does each pending case decline?

`ScoreCase` (`model/feature/occtparity/scoreboard.go`) computes each record's outcome and then
**discards** `pf.Health().Reason`. That is correct for scoring — the scoreboard is a tally, not a
diagnosis — but it means the pending population is opaque without re-running the engine. This probe
replicates `ScoreCase`'s traversal and keeps the reason, plus a per-pick edge signature
(curve kind / `ops.ClassifyEdgeConvexity` / adjacent surface kinds / CLOSED) and the area delta.

It is **overlay-mounted and must never be committed into `model/feature/occtparity/`** — it is not a
gate, it asserts nothing, and a `zz_`-prefixed always-present test file would just slow the suite.

## Running it

Write an overlay file mapping the probe into the corpus package (absolute paths required by
`go test -overlay`):

```json
{"Replace":{
  "<REPO>/model/feature/occtparity/zz_scout_probe_test.go":
  "<REPO>/test-utilities/occt-blend/scout/zz_scout_probe_test.go"
}}
```

Then:

```sh
SCOUT_OUT=/tmp/probe.tsv go test ./model/feature/occtparity/ \
  -run TestZZScoutFaultyProbe -count=1 -timeout 40m -overlay /tmp/overlay.json
```

The probe skips unless `SCOUT_OUT` is set, so an accidental overlay-less run costs nothing.

**Cost: ~121 s uncontended** for all 475 records — cheap enough to be a standing gate on every
merge-train landing. (The ~1,750 s full-suite cost is the per-case gate tests and honest meshes,
not the scoreboard pass.)

## Output format

Tab-separated, one line per corpus record, no header:

| col | field |
|---|---|
| 1 | grid (`simple`, `complex`, `bfuseblend`, `tolblend_simple`, `encoderegularity`) |
| 2 | case |
| 3 | outcome (`PASS`, `PASS(deviation)`, `FAIL(faulty)`, `FAIL(area)`, `SKIP(...)`) |
| 4 | pick count |
| 5 | per-pick edge signature |
| 6 | **decline reason** — the field `ScoreCase` discards |
| 7 | area delta vs the corpus record |
| 8 | validity conjuncts, for faulty cases with an empty reason |

## Captured census

`pending-census-2026-08-01.tsv` is the full 475-record sweep taken at
`feat/occt-blend-parity-corpus` tip `09a9b2d1`, the state at which the fillet effort was suspended.
It backs Appendix A of `architecture/audits/fillet-occt-parity-audit-2026-08.md`:

- 148 green all-grid (144 PASS + 4 PASS(deviation)), 132 green in the simple grid
- 99 `FAIL(faulty)`, 5 `FAIL(area)`, `SkipQuarantine` 0
- 108 SKIP(varradius), 61 SKIP(import), 54 SKIP(todo)

Useful one-liners against it:

```sh
# outcome census
awk -F'\t' '{print $3}' pending-census-2026-08-01.tsv | sort | uniq -c | sort -rn

# pending cases grouped by decline reason (digits normalised)
awk -F'\t' '$3=="FAIL(faulty)"{print $6}' pending-census-2026-08-01.tsv \
  | sed 's/[0-9]\+/N/g' | sort | uniq -c | sort -rn
```

Regenerate rather than trust it once the branch moves — it is a snapshot, not a fixture.
