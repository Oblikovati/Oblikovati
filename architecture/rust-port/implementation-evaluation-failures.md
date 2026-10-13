# Typed prism evaluation failures — Q3 / partial K03

The prism adapter now returns `EvaluationFailure` through both model evaluator
implementations. Feature replay, session adoption, and desktop scene health
retain the same owned failure. The viewport accepts the evaluator's failure
type without converting it to text or changing which bodies are displayed.

Each failure records its phase, failure class, affected authored entity or
result role, and available feature/sketch identities. `Error::source` preserves
the native Martensite `Failure` or model `SketchError`. Native exception type,
message bytes, and stack survive cloning. A failed validity checker is distinct
from a completed checker that reports invalid geometry. Missing or ambiguous
history remains a separate incomplete-history refusal.

The existing geometry and replay path is unchanged. Profile construction,
prism construction, validation, copy, history, and analytic measurements all
retain their current native calls and options. Shared shape checks and copying
now have one typed implementation; existing text-only probe entry points render
its result at their boundary. User-facing messages and stored-body debug health
messages remain compatible. The failure's `Debug` presentation deliberately
keeps the previous quoted text; consumers use typed accessors and `source()`
for diagnostics.

## Verification cases

- Null-shape validation and copying retain their actual native exceptions.
- Invalid geometry has no fabricated native exception and reports validation
  failure separately from checker execution failure.
- A collinear profile declines at validation, with the same legacy message.
- Incomplete and ambiguous face history retain the input edge or diagnostic
  face index. These indices are not persistent names.
- Broken authored boundaries retain feature and sketch context through replay.
  Independent bodies still finish, and corrected inputs recover.
- A profile width of `1e-15 cm` reaches the kernel and collapses edge 101.
  Desktop health retains phase `Profile`, class `InvalidGeometry`, feature 1,
  and the source sketch. No old body is displayed. Restoring `4 cm` recovers.
- The native display test repeats that failure and recovery. The failed frame
  has no solid pixels; the recovered frame equals the original frame byte for
  byte. Native window pixels and title are checked.

## Scope and remaining work

This changes the internal application adapter and evaluator contract. It does
not add a public automation DTO or change the API/kernel release pins. The
application still consumes Martensite v0.2.1 at
`193572dc3665598c4948777c87375bad7dc8114a` and API v0.158.0 at
`32ff38398ab6d6a7df0ca838654d5b8bc8d8b9cd`.

Ordered full validation, measured tolerance, complete entity history, general
analytic region construction, and cancellation remain open. Existing naming
and body-reference services still own text errors; this change classifies a
naming refusal but cannot recover a typed cause that those services do not
supply. Display mesh errors and unrelated adapter operations still use their
existing text interfaces. No missing validation level is newly claimed as
passed. Follow [kernel readiness](martensite.md) and
[region evidence](implementation-profile-regions.md) for those limits.

## Native test environment

The first gate attempt used a 1280×960 Xvfb screen. The application opens a
1440×900 window, and X11 `GetImage` returned `BadMatch`. A single scene test
reproduced the failure. The same test passed on a 1920×1080 screen. The clean
wave base `8db35fba` also reproduced the small-screen failure.

A separate concurrent head-suite run from that clean base on the larger screen
passed 111 tests and failed six native pixel/deadline checks. This verifies that
concurrent shared-display acceptance was already incomplete at the wave base.
The final gate uses the documented local configuration: 1920×1080 Xvfb,
`RUST_TEST_THREADS=1`, lavapipe, and validation layer 1.4.321. No test assertion,
modelling limit, pixel comparison, or timeout was relaxed. Concurrent native
acceptance remains open as recorded in the [naming evidence](implementation-named-prism.md).

## Final local receipt

`make gate` passed for the complete discovered workspace: all 13 crates,
all-target/all-feature builds and tests, documentation tests, formatting,
Clippy, Markdown/link checks, SPDX, workflow checks, coverage, duplication, and
`git diff --check`. The native typed-failure test passed in that run. Strict
Rustdoc for the adapter also passed.

Every package exceeds 80% production line coverage. The adapter is at 88.48%
(1474/1666 lines); the new error type and constructors are at 100%. The three
new production modules together cover 141/161 lines (87.58%). Workspace Rust
duplication is 0.70%. Review preserved call order, native options, source comments,
message rendering, and platform-independent IDs. New numeric values are
regression inputs, not modelling-policy constants. Linux is the executed
platform; Windows/macOS and concurrent native display acceptance remain open.

No kernel source, public automation contract, or release pin changed. Full K03
validation and the overall Rust port remain incomplete.
