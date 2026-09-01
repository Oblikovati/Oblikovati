# Testing 03 — Test tiers and change-based selection

*How the suite stays fast enough to run on every edit. Implements the measurement
below; the pyramid it serves is [testing/README](README.md).*

## The measurement (2026-09-01, 32 cores, `CGO_ENABLED=0 go test -count=1 ./...`)

| | Before | After |
|---|---:|---:|
| Full suite, wall clock | **14 min 31 s** | tier 1: **20 s** / tier 2: see below |
| Total test time | 3061 s (51 min) | unchanged — the work is the same |
| Packages | 128 | 128 |
| Test functions | 10 440 | 10 440 |

The suite was never slow because it has 10 000 tests. It was slow for three reasons,
and each has a rule below.

**1. The cost is not spread.** 151 of 9644 tests held 89 % of the suite's time; 7912
tests finished in under 0.1 s. Twenty tests alone were 63 %.

**2. The slow packages ran on one core.** Go runs packages in parallel but tests
inside a package in series unless they call `t.Parallel()`. `kernel/ops` burned 190 s
on one core with 31 idle. `model/feature` burned 864 s.

**3. There was no way to select a subsystem.** `kernel/ops` is 120 000 lines in one
package, `app` 85 000. A subsystem is not a package, so `go test ./...` was the only
honest command.

## Rule 1 — two tiers, split by measured cost

| Tier | Command | Contents | Budget |
|---|---|---|---|
| **1 — unit** | `make test` (`-short`) | everything that is not corpus or oracle | seconds |
| **2 — corpus** | `make test-corpus` | corpus, oracle, real-file and parity tests | minutes; before a push |

**A test that takes 2 s or more in a SEQUENTIAL run belongs to tier 2.** Guard it as
its own first statement:

```go
func TestCoilJoinFinePitchWatertight(t *testing.T) {
    if testing.Short() {
        t.Skip("corpus tier (~224s): `make test-corpus`")
    }
    ...
```

Measure with `make test-slowest-serial`, never with a parallel run: under
`t.Parallel()` a 0.05 s test on a busy core reports seconds while its package still
finishes fast, so per-test elapsed is not a budget.

A package that is a corpus **in full** gates itself once, in `TestMain`, rather than
in 154 separate tests — see `model/feature/occtparity/tier_test.go`. One gate cannot
drift out of step with the package, and a test added later inherits the tier instead
of having to remember it.

`testing.Short()` is the mechanism, not a build tag: a tagged file is invisible to
`go build`, `go vet` and `golangci-lint` by default and rots. Every tier-2 test still
compiles, vets and lints in the ordinary run.

## Rule 2 — a hot package runs its tests in parallel

Every top-level test in the 13 packages that dominated the measurement calls
`t.Parallel()` as its first statement. Kernel operations are pure functions over
immutable inputs and `app` tests each build a fresh `Session`, so they are
independent by construction.

Two classes of test must stay **sequential**, and are excluded:

- **measurement tests** — `testing.AllocsPerRun`, per-frame timing budgets, memo-hit
  counters. Another test on the same core changes the number they measure.
  (`kernel/geom/allocation_test.go`, `kernel/ops/*_bench_test.go`,
  `app/raypicker_perf_test.go`.)
- **tests that mutate package or process state** — `kernel/ops/boolean_analytic_guard_test.go`
  writes `curvedGuardBracketOverride`; `app/help_router_test.go` sets environment.

Sequential and parallel tests never overlap: `go test` runs the sequential ones to
completion first, then resumes the paused parallel ones together. Leaving a test out
of the parallel set is therefore always safe.

## Rule 3 — test what the change can reach

`make test-impacted` runs tier 1 on only the packages the current change set can
affect. It maps changed files to their packages, then widens that seed over the
**test-inclusive** import graph (`go list -test`), so a package that reaches a change
only through a `_test.go` import is still selected.

```
make test-impacted                      # vs origin/develop
make test-impacted IMPACT_BASE=HEAD     # only the working tree — what the git hook runs
make test-impacted-corpus               # the same selection, tier 2
```

A change to `go.mod`, `go.sum`, `go.work`, the `Makefile`, `.golangci.yml` or
`tools.go` selects everything: those inputs can change how any package builds.
A change no package owns (documentation) selects nothing.

Selection is a convenience, not a gate. `kernel/geom` is imported by 49 of 128
packages and `math` by 77, so touching the low layers still selects most of the
module. What bounds the cost there is rules 1 and 2, not the selection.

## Rule 4 — the tier-1 promise is gated without running anything twice

Tier 2 is a **superset** of tier 1, so a CI job that runs `-short` on the same OS as
the full run re-runs ~9400 tests for nothing. CI therefore has no tier-1 job. The
gate reads the tier-2 JSON that the Linux leg already writes, and fails on any test
that was slow **and does not guard itself**:

```
go run ./cmd/testslowest -top 25 -unguarded-budget 60 -module-root .
```

Guarded-ness is decided by a static scan of the source
(`test-utilities/testguard`), which follows a guard one call deep into a harness —
`model/feature`'s OCCT tests reach `testing.Short()` through `runCorpusGrids`, so a
direct-body check would misread them. Cross-checked against a real `-short` run:
9665 of 9669 tests agreed, and the four that did not had been added between the two
runs.

The limit is deliberately loose. Elapsed time in tier 2 is stretched by core
contention, so guarded and unguarded tests are separated by a wide margin, not a
tight one: the slowest honest test measured **22 s** against **40–434 s** for the
corpus on 32 cores. 60 s sits in that gap.

A CI runner has four cores and adds coverage instrumentation, which stretches
everything roughly threefold and narrows that gap — the first run on #3496 put the
corpus at 67–1166 s with the gate still green. So the gate reports the **slowest
unguarded test and its headroom on every run, pass or fail**: a budget that only
speaks when it breaks cannot warn you that it is about to. Read that line before
raising the limit. It catches the class that actually hurt — a 100 s+
corpus test with no guard — and does not resolve a 2 s-class one.

For the tight gate, `make test-budget` fails when a tier-1 **package** exceeds
`TIER1_PACKAGE_BUDGET` (90 s). It costs a `-short` run, so it is a local command, not
a CI job. The budget is per package, not per test, because package wall time is the
only figure that does not move with how the tests inside were scheduled.

## What runs where

| Gate | When | Runs |
|---|---|---|
| pre-commit hook | every commit | tier 1 on the impacted packages |
| CI `test` | every PR, 3 OSes | tier 2; the Linux leg adds coverage and the guard gate |
| CI `head` | every PR | the head module only — a separate module, no overlap |
| CI `race` | push to `release` | tier 2 under `-race`, corpus skipped |

Nothing in that table runs the same test twice on the same platform. The three OS
legs run the same tests deliberately: byte-identical output across platforms is a
kernel ground rule, so it has to be checked on each.

`make ci` runs `fmt-check vet lint cover` — one suite run, not three. `make ci-race`
adds the race detector for a release.

**Both modules, every time.** `head/` is a separate module, so `go build ./...`,
`go vet ./...` and `go test ./...` from the repo root do not compile one file of it. A
change that renames or moves a kernel symbol passes every root-level check and still
breaks the four head CI jobs — which is what the `kernel/ops/tessellate` extraction did.
`make vet` and `make lint` both descend into `head/` for that reason; running
`golangci-lint run` or `go vet ./...` directly does not, and is how the gap opens.

## The kernel/ops split (#2183)

`kernel/ops` was 120 000 lines in one package, so a change anywhere in it re-ran every
test it owned and rebuilt the 300+ packages that imported it. It is now packaged BY
OPERATION, as the kernel ground rules require:

| Package | Source files | Test files | What it owns |
|---|---:|---:|---|
| `kernel/ops` | 9 | 14 | the façade, plus section/shell/split/thicken |
| `kernel/ops/blend` | 222 | 190 | fillet, chamfer, draft — one engine (ADR-0050/0051) |
| `kernel/ops/boolean` | 23 | 88 | classification, curved pairs, CSG, mesh arrangement |
| `kernel/ops/tessellate` | 55 | 65 | the tolerance-driven mesher |
| `kernel/ops/query` | 19 | 22 | pick, mass properties, boxes, containment |
| `kernel/ops/surface` | 17 | 14 | extend, offset, replace, rebuild, fair, untrim |
| `kernel/ops/heal` | 16 | 21 | sew, stitch, snap, cap, fill, drop, voids |
| `kernel/ops/validate` | 8 | 10 | the ordered validity levels |
| `kernel/ops/transform` | 5 | 6 | move, deform, re-surface |

Shared substrate sits below all of them in `kernel/ops/internal`: `mesh` (the Mesh type
and the point welder), `probe` (read-only geometric questions), `retopo` (rebuild
helpers), `tol` (model-relative tolerance constructors) and `disjoint` (union-find).

### The method

A subpackage can only be carved where the symbol edge is ONE-WAY, because Go rejects the
cycle otherwise. Every extraction followed the same three steps:

1. **Measure, do not guess.** A small AST analyser reports the symbols crossing a proposed
   group boundary in each direction. Skip `SelectorExpr.Sel`, composite-literal keys and
   field names, or field accesses read as package symbols and the numbers are fiction.
   Locals shadowing a package-level name still produce false edges — `weld` and `cut` both
   did — so a one-symbol edge is worth reading before acting on it.
2. **Move the shared substrate DOWN, not sideways.** `validate`'s outgoing set fell from
   seven symbols to two that way; `blend`'s from 34 to 9. Exporting a symbol to reach it
   across the new seam is the move that makes the next extraction impossible.
3. **Fix the layer that owns it.** Four things blocked `heal`, and none was solved by an
   export: the boundary iso-curves are a property of a B-spline surface, so they became
   `geom.BSplineSurface.BoundaryUIso`/`BoundaryVIso`; `firstNurbsFace` is a read-only
   question, so it became `probe.FirstNurbsFace`; `fullDomainBody` and the planar-loop soup
   builder are re-topology, so they became `retopo.FullDomainBody` and
   `retopo.PlanarLoop`/`BuildSolidFromLoops`.

The direction is now pinned by archguard's edge allowlist, which splits `kernel/*` and
`kernel/ops/*` one level deeper than it did (#2194): every edge between families is a declared
row, and no family may name the `kernel/ops` façade — the façade forwards to them, so that
edge is a cycle waiting to happen. Seeding the rows from the real graph also made the two
known inversions visible where they are enforced: `kernel/exchange` and `kernel/hlr` both sit
above the operation layer but live inside `kernel/`, and carry a TODO with the issue that
moves them (#2195, #2196).

### What it bought

`kernel/ops` ran one test binary of ~1550 tests, so every change to any operation re-ran all
of them. The families now hold 813 (blend), 231 (boolean), 208 (tessellate), 81 (query), 59
(the façade), 55 (heal), 52 (validate), 47 (surface) and 19 (transform): a fillet change no
longer runs the boolean suite, and neither runs the other's.

Package SELECTION improves less, and it is worth being precise about why. A change under
`kernel/ops/heal` selects 31 of 148 packages where anything in `kernel/ops` used to select 51;
a change under `blend`, `boolean` or `query` still selects 51, because `model/feature` imports
all three and `app` imports `model/feature`. Narrowing that is a different cut — splitting
`app` and `model/feature` — not this one.

### Two seams the split forces

**Test fixtures.** Go does not share `_test.go` helpers across packages, so a fixture two
families both need must become real code. `test-utilities/brepfixture` is that home — plain
constructors over `geom`/`topo`/`subd` with no operation and no assertion in them, so they
cannot make a test pass for the wrong reason. It absorbed the copies of `quadBody`,
`cubeFaces`, `tetra`, `shellBox`, `topFaceKey`, `verticalEdgeKey` and the #2009 bunched
strip that had accumulated in four packages. Fixtures that need an operation to build them
live in `test-utilities/opfixture` instead, because `kernel/ops/validate`'s own tests use
`brepfixture` and `validate` sits below the operations — one package importing an operation
would close a cycle.

**Where a test lives.** A test follows the code it exercises, not the file it was in. Two
that had drifted: `point_in_solid_test.go` was covering `query`'s winding number AND
`boolean`'s `PointInsideBody`, and `analytic_oracle_test.go` mixed `query` internals with
cases whose operand only the general boolean driver can build. Each split along that line.

**The public seam.** `boolean` keeps ops-level forwarders because ~1100 call sites name
`ops.Boolean`/`ops.Cut`; `heal` (61 sites), `query` and `surface` (88) do not, so consumers
name the narrow package and stop rebuilding when an unrelated family changes.

## What this does not fix

The other large packages are still too large to be a selection unit: `app` (85 000 lines),
`model/feature` (64 000), `addin/router` (40 000).
