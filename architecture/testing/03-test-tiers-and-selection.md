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

## Rule 4 — the tiers hold their budget

`make test-budget` fails when a tier-1 package exceeds `TIER1_PACKAGE_BUDGET`
(90 s). A new corpus test that forgets its `testing.Short()` guard costs 100 s or
more, so the gate catches it; the slowest honest tier-1 package sits near 15 s on 32
cores. CI runs this on the Linux leg.

The budget is **per package**, not per test, because package wall time is the only
figure that does not move with how the tests inside were scheduled.

## What runs where

| Gate | When | Command |
|---|---|---|
| pre-commit hook | every commit | `make test-impacted IMPACT_BASE=HEAD` |
| CI `test` | every PR, 3 OSes | tier 1, plus `test-budget` on Linux |
| CI `test-corpus` | every PR, Linux | tier 2 with coverage |
| CI `race` | push to `release` | tier 2 under `-race` |

`make ci` runs `fmt-check vet lint cover` — one suite run, not three. `make ci-race`
adds the race detector for a release.

## What this does not fix

The packages are still too large to be a selection unit: `kernel/ops` (120 000
lines), `app` (85 000), `model/feature` (64 000), `addin/router` (40 000). Splitting
them by operation is what the kernel ground rules already require ("Package by
operation, not by case: split `kernel/ops` into boolean, blend, tessellate, validate,
heal, query") and is tracked under **#2183** with its children — #2207 (query), #2209
(tessellate), #2210 (transform), #2211 (validate), #2194 (archguard one level deeper).
Until those land, selection is package-shaped and a kernel edit still runs the whole
kernel.
