# Native Chamfer diagnostic (#4046)

On 2026-11-17 the live `viewport-feature-picking` scenario **Select an edge to
preview a chamfer** passed all seven steps under Xvfb and software Vulkan. The
ribbon and viewport clicks and the distance edit used X11 input. The test
checked the selected edge, visible preview, a changed preview after a keyboard
distance edit, and the committed `Chamfer1` browser entry. The live feature
SHA-256 was `5a2d3067d4e627b30f1e2ed004ce27b753f2934fdb33679166933f5e0d8eb5fc`
before and after the run.

A separate, temporary `cancel-regression.feature` used native clicks to cancel
a visible Chamfer preview and check that the original solid remained. The
existing Fillet preview scenario also passed after sharing the dialog. The
compressed reports and logs distinguish these tests from the live scenario.

These runs are diagnostic, not verified BDD acceptance: the live source is
unpinned, its fixture is unresolved, and build provenance remains open (#3978).
The common extrusion fixture sketch has four degrees of freedom. The model
capture still shows the dark-face issue #4044. These tests did not measure
frame rate or prove the 60 fps target. Chamfer modes other than equal-distance
and concave-edge behavior remain unverified.
