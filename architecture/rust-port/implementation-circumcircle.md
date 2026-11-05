# Owned circumcircle construction — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Consumers: three-point circle, arc, and arc-slot constructors in Go
`model/sketch/construct.go` and `model/sketch/composite.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The application Martensite adapter now exposes `construct_planar_circumcircle`.
It calls the released `gce::MakeCirc2d::through_points` builder and returns only
owned center, radius, ordered input residuals, and the supplied points' winding.
No mutable kernel object escapes. The application does not copy Go's determinant
formula or add a second circumcircle engine. Martensite v0.2.5 and API v0.158.0
remain pinned; no new kernel release is needed for this adapter.

The released exact `orient2d` predicate validates finite coordinates and rejects
exact collinearity, including repeated points. Its sign determines the arc seed's
winding. This classifies the supplied binary64 points only. Native construction
can still refuse noncollinear input because of its arithmetic and conditioning.
Such refusal does not change the exact classification or trigger another engine.

The caller supplies a finite nonnegative maximum point residual. Zero requests
zero measured residual; there is no default, widening, normalization retry, or
coordinate perturbation. The adapter checks finite center and positive finite
radius, then measures each input point's radial residual. Every residual must be
finite and no larger than the supplied bound. Values remain in the caller's
coordinate units. These measured binary64 residuals are not interval bounds or a
certificate of center error for ill-conditioned input. The result is a seed for
explicit sketch authoring, not proof of a valid topological profile.

The mapped native builder can report success with its default `RealLast` circle
when its bisectors are judged parallel. The adapter rejects that exact native
sentinel before metric acceptance, even with a maximum-finite caller bound.
It also refuses the zero-radius result that native arithmetic can produce for
very small points. These checks describe the released builder; no additional
absolute geometric tolerance is introduced. Near-collinear and extreme cases
remain explicit failures when native construction cannot supply an accepted seed.

`PlanarCurveFailure` keeps invalid input, native failure, and invalid result
distinct. Every failure names the three supplied points and options. Native
exceptions retain their original typed `Error::source`. The shared context helper
also preserves the existing intersection error messages. A failed call cannot
modify an earlier returned value or publish a previous successful result.

Nine focused tests cover exact cardinal circles, both windings, exact collinearity,
repeated points, every nonfinite coordinate position, invalid residual bounds,
strict versus nonzero residual acceptance, repeated calls, native sentinel/zero
radius refusal, overflow/underflow, and a tiny triangle whose rounded center
causes a native zero-norm exception. Thirty generated cases cover all six point
permutations at five binary scales. Their metric comparisons have explicit
rounding bounds; winding remains exact. An initial zero-rounding assumption for
translated generated circles was invalid and was corrected in the test only.

Four committed cases come from a standalone native OCCT 8.0.0 capture, including
an irregular triangle and a large translation. The kernel's mapped source baseline
remains its pinned revision; this installed OCCT version is an independent oracle,
not a replacement source pin. Replaying the capture produces identical bytes.
The corresponding three pinned Go constructor tests pass independently.

The capture source is `crates/martensite/tests/circumcircle/occt_oracle.cpp`.
Compile it with the installed OCCT include and library paths, linking
`TKGeomBase`, `TKMath`, and `TKernel`; run it with the OCCT library directory on
`LD_LIBRARY_PATH`. Rust tests consume the committed JSON and do not invoke OCCT.

This step does not wire the three-point constructors into `sketch.addEntity`.
The next application step must inject this capability into authoring, preserve
input point order and endpoint ownership, validate the final solved geometry,
and prove undo/reload/native consumption. Three-point arc slots also need their
complete composite ownership path. Those consumers and full R3 remain open.

The full local `make gate` passed for all 13 application crates, all targets and
features. It includes formatting, Clippy, build, tests, documentation tests,
coverage, Markdown/link checks, SPDX, workflow lint, and duplication. Existing
native tests ran serially on a 1920×1080×24 Xvfb display with software Vulkan
and validation layer 1.4.321. This is regression evidence for the application;
there is no native three-point authoring consumer in this step.
All crates exceeded 80% line coverage. The new adapter reached 80/80 lines and
the shared failure module 51/51 (both 100%). Martensite adapter coverage was
91.17%. Duplication was 0.76% (527/68,891 scanned lines), with no new clone.

The cross-platform and comment review found no new production tolerance literal,
fallback algorithm, unordered result assembly, or platform-sized identity.
The native-sentinel guard names its upstream cause. Windows and macOS runtime
verification remains open. This change is local only; no push or release.
