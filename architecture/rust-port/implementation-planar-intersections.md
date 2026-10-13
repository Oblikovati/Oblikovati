# Released planar intersections — prerequisite evidence for R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Dependency: Martensite v0.2.1,
`193572dc3665598c4948777c87375bad7dc8114a`.

The `planar_intersections` integration target tests the released general 2D
curve intersector. TKGeomAlgo is a test-only direct dependency with the same
exact pin as the other kernel packages. No production kernel dependency,
application intersection algorithm, or public API is added.

Ten tests cover crossing lines; ellipse crossings, tangency, and separation;
a trimmed circular quadrant with interior, endpoint, and excluded crossings;
a rational quadratic circular arc; a cubic with three roots; a quadratic
tangent root; bounded same-direction overlap; unbounded opposite-direction
overlap; incomplete counts; and a failed reinitialization after success.
Point cases run in both operand orders. Their checks require completion before
reading results, check the number of roots and overlaps, and compare each
reported point against an independent analytic expected position. Both native
parameters are evaluated on their own input curves and must reproduce that
position. The bounded overlap endpoints have the same incidence checks.
Test sorting removes kernel enumeration order from the acceptance condition.

The cubic has `x(t) = t` and `y(t) = (t - 0.2)(t - 0.5)(t - 0.8)`.
The quadratic tangent has `x(t) = t` and `y(t) = (t - 0.5)^2`.
The weighted quadratic is the standard radius-two circular quadrant with
weights `[1, 1/sqrt(2), 1]`. These formulas provide the root positions without
using another call to the intersector as the expected result. The fixture
intersection tolerance is 1e-9 and point comparison bound is 1e-8 in fixture
coordinates. Neither value defines application tolerance policy.

Two wrapper behaviors require explicit handling in the future application
adapter. First, the high-level counts return zero before computation is done;
the underlying fallible count methods report an error. Zero alone therefore
does not prove a completed empty intersection. Second, reinitialization with
a null input returns an error but retains the previous completed result. Its
completion flag then describes the old call. A caller must propagate the
error and discard that result. A fresh immutable call boundary can avoid
reuse. The regression records released behavior; it does not endorse stale
results or require a future kernel release to retain this behavior.

This is bounded release-integration evidence. Successful completion and small
point residuals do not certify root completeness for arbitrary curves,
exact topological decisions, or validity across an overlap. The initial
incomplete object is a wrapper-contract test, not a reproduced solver failure.
Periodic seams, offset curves, general spline construction from authored fit
points, extreme scales, generated near-degenerate inputs, transition mapping,
and production tolerance and failure policies remain open. There is no new
application trim or extend command. Stable edit identities, constraints,
feature history, persistence, undo, and native presentation of those commands
still require implementation and tests.

Run the focused checks with:

```sh
cargo test --locked -p oblikovati-martensite --test planar_intersections
```

The full local `make gate` passed for all 13 crates in the discovered workspace:
all targets/features, formatting, Clippy, build, coverage tests including native
Linux presentation under isolated Xvfb, doctests, documentation/link/SPDX checks,
workflow lint, and duplication. All crates exceed 80% line coverage; the
Martensite adapter has 88.48% (1,474/1,666). Production-code duplication is
0.76% (510/67,096 lines); the configured scan excludes tests. Review found no
production tolerance change, new platform-specific constant, persistent-ID
change, or dependency version change. These tests do not provide cross-platform
runtime or full R3 acceptance evidence.
