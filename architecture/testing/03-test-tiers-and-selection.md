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

## What this does not fix

The packages are still too large to be a selection unit: `kernel/ops` (120 000
lines), `app` (85 000), `model/feature` (64 000), `addin/router` (40 000). Splitting
them by operation is what the kernel ground rules already require ("Package by
operation, not by case: split `kernel/ops` into boolean, blend, tessellate, validate,
heal, query") and is tracked under **#2183**.

`kernel/ops` now has five neighbours where it had none — `transform` (#2210),
`validate` (#2211) and the `internal/mesh`, `internal/probe`, `internal/retopo`
leaves — so those tests are their own binaries. The remaining six extractions are
blocked on entanglement that a move cannot fix, measured rather than guessed:

| Group | Files | Blocked on |
|---|---:|---|
| blend | 230 | the tessellation core it meshes through |
| tessellate | 48 | 43 symbols across 18 files, mutual with blend and validate |
| query | 18 | reads TESSELLATED data for mass properties (#3420) — a ground-rule violation, not a move |
| boolean, surface, heal | — | follow the three above |

The rule that decides all of them: a subpackage can only be carved where the symbol
edge is ONE-WAY, because Go rejects the cycle otherwise. What made `transform` and
`validate` possible was moving shared substrate DOWN into `internal/` leaves first —
`validate`'s outgoing edge set fell from seven symbols to two that way. The same
method applies to the rest, but `query` needs #3420 resolved before it can move at
all: it is phase-2 algorithmic work, outside #2183's refactor-only phase 1.
