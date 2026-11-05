# Released Martensite v0.8 adoption

Tracking: [#3809](https://github.com/Oblikovati/Oblikovati/issues/3809).

The current Martensite dependency graph targets the
[v0.8.0 release](https://github.com/Oblikovati/Martensite/releases/tag/v0.8.0),
commit `59aca38960bdfbaa75a07a80d9b2de522005ddac`. The source guard checks
that exact revision for every locked Martensite package. Toolkit versions remain
the versions authored by that release: Vulkan is 0.7.1, TKV3d is 0.7.0,
TKService is 0.4.1, TKernel is 0.4.2, and TKMath is 0.3.6. The lock update changes
only this graph.

The initial renderer intake built against the release. Its 22 native viewport tests passed,
including palette changes, sRGB and alpha, point markers, preview clear,
scene refresh, camera navigation, native surface capture and close. Live OS input
created a parameter-driven rectangle, verified preview and committed colours,
switched themes, saved and reopened it in a fresh process. MCPBridge confirmed
healthy DOF=0 geometry, and both desktops closed normally. The integrated host at `bf25550a` later passed the full local gate and shared
browser/API selection acceptance in #3812. The authored-coordinate fix is
included; its tracking issue is #3808.

The release adds TKBO, TKBool, TKFillet and TKCaf. Available toolkits do not by
themselves prove Boolean, blend, feature naming or feature replay acceptance.
Those operations must pass their host contracts and independent result gates.

The sketch geometry bindings now use v0.8 as well: TKG2d is 0.3.7,
TKGeomAlgo is 0.3.8, and TKGeomBase is 0.3.6. All test targets compile and
186 adapter checks, including doctests, pass with their existing numerical
assertions. The retained OCCT and Go construction corpora remain unchanged.
The two direct legacy 2D dependencies are removed. The full local gate passed
with all 13 crates above 80% line coverage and 0.86% duplication. Live OS
keyboard/mouse input created the parameter-driven XY rectangle with DOF=0,
changed size, ran Undo/Redo, saved, closed, reopened, and reopened in a fresh
process. MCPBridge selection queries matched the rendered highlight. Both test
desktops closed normally. Hosted platform qualification remains required.
Tracking is [#3814](https://github.com/Oblikovati/Oblikovati/issues/3814), under
[ADR-0029](../decisions/ADR-0029-released-sketch-geometry.md).

The production solid geometry adapter retains its qualified RustyCADKernel
v0.2.7 baseline. The unmerged solid-adapter work is preserved. Martensite
[#386](https://github.com/Oblikovati/Martensite/issues/386), mass-property
qualification, and [#390](https://github.com/Oblikovati/Martensite/issues/390),
the production analytic sketch-region contract, remain open. The validated Linux
ARM64 runtime is tracked in
[#388](https://github.com/Oblikovati/Martensite/issues/388). No runtime fallback
or second renderer is added by this dependency update. The full cube journey
remains unaccepted.

Analytic editor picking is being integrated under
[#3816](https://github.com/Oblikovati/Oblikovati/issues/3816). Its implementation
and local evidence are recorded in the
[analytic picking ledger](implementation-analytic-sketch-picking.md). Constraint
tool and full cube acceptance remain separate requirements.
