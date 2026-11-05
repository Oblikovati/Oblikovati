# Released Martensite v0.8 adoption

Tracking: [#3809](https://github.com/Oblikovati/Oblikovati/issues/3809).

The current Martensite dependency graph targets the
[v0.8.0 release](https://github.com/Oblikovati/Martensite/releases/tag/v0.8.0),
commit `59aca38960bdfbaa75a07a80d9b2de522005ddac`. The source guard checks
that exact revision for every locked Martensite package. Toolkit versions remain
the versions authored by that release: Vulkan is 0.7.1, TKV3d is 0.7.0,
TKService is 0.4.1, TKernel is 0.4.2, and TKMath is 0.3.6. The lock update changes
only this graph.

The desktop builds against the release. All 22 native viewport tests pass,
including palette changes, sRGB and alpha, point markers, preview clear,
scene refresh, camera navigation, native surface capture and close. The combined
host gate and live UI acceptance remain required before this branch is pushed.
The separately tested authored-coordinate fix is tracked in #3808.

The release adds TKBO, TKBool, TKFillet and TKCaf. Available toolkits do not by
themselves prove Boolean, blend, feature naming or feature replay acceptance.
Those operations must pass their host contracts and independent result gates.

The production geometry adapter still uses its qualified RustyCADKernel v0.2.7
baseline. The unmerged Martensite adapter work is preserved. Martensite
[#386](https://github.com/Oblikovati/Martensite/issues/386), mass-property
qualification, and [#390](https://github.com/Oblikovati/Martensite/issues/390),
the production analytic sketch-region contract, remain open. The validated Linux
ARM64 runtime is tracked in
[#388](https://github.com/Oblikovati/Martensite/issues/388). No runtime fallback
or second renderer is added by this dependency update. The full cube journey
remains unaccepted.
