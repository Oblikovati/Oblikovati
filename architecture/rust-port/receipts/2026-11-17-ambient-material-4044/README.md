# Ambient material diagnostic (#4044)

The isolated neutral 0.7 cube test on Xvfb/Lavapipe showed visible face
values 38, 132 and 173 before the change. The dark face failed a floor of
80/255. For comparison, the Go shaded-mode shader at the default rig's
ambience and exposure produces about 90/255 from its ambient term alone for
the same 0.7 input; the test leaves ten levels for rendering differences.
This is a floor comparison, not pixel parity with the Go renderer.

The renderer's default material applies an ambient coefficient of 0.25 in
addition to the viewer's 0.18 ambient light. With the viewer light unchanged,
the host now gives its color-only triangle material full ambient reflectance.
The cube faces measured 100, 157 and 192 on Lavapipe. Red, green, blue, dim,
and reversed-face checks passed. The four focused tests passed on Lavapipe and
AMD Radeon 8060S / RADV; the full native viewport target passed 45 tests with
three ignored.

The desktop Chamfer scenario passed as a diagnostic after the change. At the
same sampled points, the old capture in
`../2026-11-17-chamfer-native-4046/chamfer-committed-4046.png` gave front,
top and side values of (38,38,39), (145,147,149), and (132,134,135). The
new `after.png` gave (100,101,102), (168,170,172), and (157,159,161).
The scenario source was live and unpinned; its SHA-256 was
`5a2d3067d4e627b30f1e2ed004ce27b753f2934fdb33679166933f5e0d8eb5fc`.
The capture is not verified BDD acceptance. This does not prove full lighting
or material parity and does not measure frame rate.
