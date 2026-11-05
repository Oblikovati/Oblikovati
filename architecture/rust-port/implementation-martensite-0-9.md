# Released Martensite v0.9 adoption

Date: 2026-11-06. Tracking: [#3821](https://github.com/Oblikovati/Oblikovati/issues/3821).

## Release boundary

The renderer and authored sketch geometry graph targets the
[v0.9.0 release](https://github.com/Oblikovati/Martensite/releases/tag/v0.9.0),
commit `b7c0f682577e842a81f37b20813451e7167ec80c`. The dependency guard requires
that one immutable source for every locked Martensite package. A regression
refuses the earlier release for renderer and sketch packages.

The release keeps Vulkan 0.7.1, TKV3d 0.7.0, TKService 0.4.1, TKernel 0.4.2,
TKMath 0.3.6, TKG2d 0.3.7, TKGeomAlgo 0.3.8 and TKGeomBase 0.3.6.
Transitive TKBrep changes to 0.3.9 and TKG3d to 0.3.8. The lock update changes
only the Martensite graph; it does not update unrelated dependencies.

The source review covers the exact v0.8.0 to v0.9.0 tag diff. New TKDE,
TKXSBase, TKVCaf and TKXCaf toolkits supply exchange and document presentation
facilities. Existing text and binary B-rep writers use Martensite headers;
readers retain the earlier headers. The existing spatial geometry change is a
diagnostic wording update. No new exchange API or feature capability is inferred
from toolkit availability.

## Qualification

The workspace compile check and full local gate passed. All 13 crates exceed
80% line coverage, with 0.89% duplication. The gate includes the 320 head checks,
native viewport tests under Xvfb, the release-source regression, formatting,
Clippy, documentation, SPDX and workflow checks.

Live X11 mouse and keyboard input with the Go MCPBridge passed analytic picking,
Horizontal/Vertical/Coincident, cancellation, undo/redo, save and fresh-process
reopen. The geometric test sketch remains healthy at DOF 2. A separate desktop
created the parameter-driven rectangle, changed size from 10 mm to 20 mm, used
undo/redo, saved and reopened the healthy DOF-zero sketch. A fresh process also
reopened it with empty transient selection. MCP screenshots were checked. All
four test desktops closed normally. Hosted platform checks remain required.

The production solid adapter remains on its separately qualified baseline.
Martensite [#386](https://github.com/Oblikovati/Martensite/issues/386), mass
properties, and [#390](https://github.com/Oblikovati/Martensite/issues/390), the
analytic sketch-region contract, remain required for solid migration. Linux
ARM64 Vulkan qualification remains tracked in
[#388](https://github.com/Oblikovati/Martensite/issues/388). The complete cube
journey and dimensional UI remain unaccepted. No fallback renderer is added.
