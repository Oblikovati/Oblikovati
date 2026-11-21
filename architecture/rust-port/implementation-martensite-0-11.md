# Released Martensite v0.11 adoption

Tracking: [#3847](https://github.com/Oblikovati/Oblikovati/issues/3847).

## Release boundary and review

The renderer and authored sketch graph targets
[v0.11.0](https://github.com/Oblikovati/Martensite/releases/tag/v0.11.0), peeled
commit `4141137449d911884c9a3fd261f7913498548e90`. The annotated tag object is
`3355085afe8a4630b9a93aa197fea9b586ac8fc5`. All 15 linked packages use the same
immutable source. The other 184 locked packages are unchanged, including the
separately qualified RustyCAD solid adapter. No API dependency changed.

The full upstream diff contains 176 files. Its changed consumed runtime code is
in TKShHealing: periodic seam alignment, reversed-face wire orientation,
second-pass face ownership, adjacent crossing tolerance bookkeeping, loop repair
and inserted-edge iteration. TKShHealing changes from 0.3.7 to 0.3.8. The other
consumed component versions remain unchanged. Vulkan and its validation
installer are unchanged from the previous pin.

The new TKDESTL, TKMESHVS and TKXMESH packages remain unlinked. Their publication
does not establish an application import/export or meshing feature contract.
Review covers the full file inventory and changed consumed source and tests; it
does not claim full review of those new unlinked implementations.

## Verification plan and initial evidence

The viewport build and source-pin tests passed. The source regression now refuses
v0.10.0 for renderer and sketch packages. All ten new upstream healing regressions
passed from the released source checkout, with the host target and bounded Kache
storage reused. The host gate does not run dependency unit tests automatically,
so this is separate evidence for the changed healing code.

Qualification must run the full local gate and platform compile checks. Native
X11 acceptance uses OS input for a parameter-driven rectangle at DOF zero, size
edit, undo/redo, save/reopen, and current Line/Format tools. MCPBridge reads and
captures remain read-only; inspect actual PNGs and close owned desktops.

The ARM64 validation installer is still missing. The proposed LunarG ARM64 file
at the pinned x64 package version returns HTTP 404. This is not proof of a working
alternative. [Martensite#388](https://github.com/Oblikovati/Martensite/issues/388)
still requires a qualified runtime and a named extension admission error. Keep
validation enabled and all native tests active. Analytic sketch regions
([#390](https://github.com/Oblikovati/Martensite/issues/390)) and the complete
cube/Extrude/named-face journey remain unqualified.

## Local qualification

The full `make gate` passed all 13 crates and all 385 head tests on Xvfb.
Coverage is 91.94% to 99.60% (GUI 98.25%, head 92.74%). Duplicated lines are
1.03% (930 of 90,676 lines). Formatting, Clippy, Markdown, SPDX and workflow
checks passed. The log is `/tmp/oblikovati-martensite-0-11-gate.log`.

Six cross-target Clippy checks passed: viewport on Windows x64/ARM64 MSVC and
macOS Intel/ARM64, plus host/GUI on Windows x64 GNU and macOS ARM64. These compile
checks do not claim native platform runtime evidence.

Native X11 acceptance created `size = 10 mm` and a fully constrained XY rectangle,
changed size to 20 mm, exercised undo/redo, saved and reopened in the same and a
fresh process. Both dimensions retain `size`; reads confirm DOF zero, two
horizontal, two vertical and one coincident constraint. Continuous Line covered
private preview, closure, Backspace, Fix, history and fresh reopen at DOF zero.
Format acceptance covered independent centerline and construction roles,
preview changes, selected conversion, cancellation, history and fresh reopen.
The reopened axis can be removed and restored without losing its identity.

MCPBridge supplied read-only checks and captures. Actual PNGs were inspected.
All six owned desktops closed normally. The theme probe was moved from a pixel
now covered by the Modify caption to uncovered ribbon background; its exact
expected RGB stayed unchanged.

The worktree and bounded Kache storage were reused. After qualification, a
source-based cleanup removed 482 obsolete v0.9/v0.10 SDK artifacts from this
worktree's target tree: 5.78 GiB logical size, with 5.77 GiB of exclusively owned
allocated blocks. Current v0.11 artifacts, RustyCAD artifacts and all unmerged
branches were retained. No new target tree was created; incremental storage
remains disabled.

The comment/coherence and hard-coded-value review completed before push. New
constants are immutable revision identities. Hosted platform checks must still
qualify the exact published commit before merge.
