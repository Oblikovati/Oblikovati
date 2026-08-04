<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OCCT blend-parity corpus (`occtparity`)

This package replays OpenCASCADE's `tests/blend/*` corpus (475 cases) against our **real
fillet feature** and asserts surface area within OCCT's own tolerance. It is the parity spec
*and* the regression gate for completing the ADR-0050 blend engine. It is test-scope only —
imported by `_test.go`, never shipped.

## Pipeline

```
OCCT tests/blend/<grid>/<case>            (unmodified OCCT test scripts)
        │
        ▼   test-utilities/occt-blend/oracle/occt_blend_oracle   (DRAWEXE + oracle.tcl)
<case>.step   +   <case>.json             (OCCT's input solid  +  picks/area/todo)
        │
        ▼   test-utilities/occt-blend/gen                        (go run)
model/feature/occtparity/corpus.json      (475 records, //go:embed)
model/feature/occtparity/fixtures/<grid>/<case>.step
        │
        ▼   RunCase  (this package)
import STEP → locate picked edges → drive AddFilletSets → Recompute → assert area
```

The oracle runs each OCCT case verbatim in a locally-built DRAWEXE and records, per pick, a
**geometric edge locator** (OCCT's own edge resolution) plus OCCT's reference area — so the
only variable under test is *our fillet*, on *OCCT's exact input geometry*. See
`test-utilities/occt-blend/oracle/README.md` for the oracle design.

## How to run

```bash
# The parity gate (hard-asserting; reds are the greening backlog):
go test ./model/feature/ -run TestOCCTBlend

# The non-gating scoreboard (per-grid PASS/FAIL/SKIP table):
go test ./model/feature/occtparity/ -run TestOCCTBlendScoreboard -v
```

## Reading the scoreboard

| outcome            | meaning                                                                    |
|--------------------|----------------------------------------------------------------------------|
| `PASS`             | our fillet built a valid solid whose area is within OCCT's tolerance        |
| `FAIL(area)`       | built a valid solid, but area disagrees with OCCT > tolerance — a parity bug |
| `FAIL(faulty)`     | the fillet was rejected / produced an invalid solid — an engine gap         |
| `SKIP(todo)`       | OCCT itself marks the case incomplete, or it is a multi-blend/loop case      |
| `SKIP(import)`     | OCCT's input did not survive STEP import, or a pick could not be located    |
| `SKIP(varradius)`  | variable-radius (buildevol) — see the follow-up below                       |

**The reds are the greening backlog and are NEVER loosened.** They are genuine ADR-0050
engine gaps: curved-neighbour fillets, corner/miter blends, n-way corners, radius-vs-maximum,
and the `FAIL(area)` parity mismatches. Closing them is separate engine work; this corpus is
the gate that stays red until they are closed. Per the milestone rule, no PR lands until the
whole corpus is green (excluding OCCT-TODO skips).

## The parameterization-invariance rule (why we match on centroid, not mid-point)

STEP import **reparameterizes every edge to `[0,1]`**. A curved edge's mid-*parameter* point
therefore does not correspond between OCCT and us (a full circle's can land a diameter away),
so the edge locator matches on the **arc-length centroid + total length** — both
parameterization-invariant, sampled with an identical 64-chord scheme in `oracle.tcl`
(`edgeloc`) and Go (`edgeCentroidLength`). Any future edge-space quantity ported from OCCT
must be reduced to a parameterization-invariant form the same way.

## Known follow-ups (tracked, not silently dropped)

- **Variable radius (`buildevol`, 108 cases → `SKIP(varradius)`).** The feature layer already
  supports variable radius (`FilletEdgeSet.StartRadius`/`EndRadius`/`RadiusPoints`). The
  blocker is that OCCT's `updatevol` law is defined in **edge-parameter space**, which STEP
  import discards — the same problem the locator solved. Porting it needs each law point's
  arc-length fraction computed in the oracle and mapped onto our reparameterized edge's `T`
  (and confirming our `RadiusPoint.T` is arc-length, not raw parameter). Until then these skip
  rather than ship a wrong law that would pollute the parity signal.
- **Unvendored fixtures (~27 cases).** Absent from OCCT's public dataset archive; listed in
  `test-utilities/occt-blend/SOURCES.md`. Their cases surface as `SKIP(import)`.
- **Fixture non-determinism.** OCCT's STEP export stamps a wall-clock timestamp, so
  regenerating the corpus rewrites every fixture's header. The committed fixtures are the
  source of truth; regenerate `corpus.json` alone (to a temp `-out`, then copy just
  `corpus.json`) to avoid churning 400+ fixtures. Normalizing the STEP timestamp in the
  generator would make regeneration clean — a possible improvement.

## Regenerating

```bash
go run ./test-utilities/occt-blend/gen \
  -occt ../OCCT/tests/blend \
  -oracle test-utilities/occt-blend/oracle/occt_blend_oracle \
  -out model/feature/occtparity
```

Runs the oracle over all 475 cases (~8 min) and rewrites `corpus.json` + `fixtures/`. See the
non-determinism note above before committing the result.
