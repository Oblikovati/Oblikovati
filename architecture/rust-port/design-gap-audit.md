# Desktop design gap audit

Source: `/mnt/vininas-projects/MCAD-Scenarios/Design/DESIGN-GAP-AUDIT.md`.
The designer audited a Windows binary on 2026-11-25. Native evidence recorded
here uses the Rust desktop on Linux under Xvfb, Openbox, and Vulkan Lavapipe.
The existing horizontal scroll fallback keeps commands accessible below the
minimum width of the compact Sketch groups.

## Verification

| Gap | Evidence or remaining work |
| --- | --- |
| GAP-01 | Native Light Mode switch, theme-aware Default backdrop, explicit High Contrast override, and a light status bar. [Default](receipts/2026-11-25-design-gap-01-4063/Light.png) · [Override](receipts/2026-11-25-design-gap-01-4063/Override.png) |
| GAP-02 | Verified content-based height, the pinned action footer, exclusive feature drafts, and Apply & Create New. [Native checks](receipts/2026-11-25-design-gap-02-05-finish-4063/README.md) |
| GAP-03 | Verified labelled Sketch groups, last-row clearance, and the emerald Finish Sketch at 1440 px. [Native check](receipts/2026-11-25-design-gap-p0-4063/README.md) |
| GAP-04 | Verified translucent origin-plane selection beneath the viewport controls, with the ViewCube kept clear and plane selection still working. [Native check](receipts/2026-11-25-design-gap-p0-4063/README.md) |
| GAP-05 | Verified upper-left placement, 40/60 fields, primary OK, inline Distance A tools, the single direction row, the profile pill, and preview toggle. Floating property windows use perimeter borders and shadows. The default distance and the remaining field and dropdown styling still need work. [Fields](receipts/2026-11-25-design-gap-05-4063/README.md) · [Extrude controls](receipts/2026-11-25-design-gap-02-05-finish-4063/README.md) |
| GAP-06 | Verified the wider File menu, command and document icons, distinct pin controls, and long-name truncation. All eight published scenarios passed. [Native checks](receipts/2026-11-25-design-gap-06-07-4063/README.md) |
| GAP-07 | Verified the filename and metadata columns, distinct type icons, active List and Grid styles, visible checkboxes, and primary action. The Home rail uses the themed wordmark and shows the active project when available. [Native checks](receipts/2026-11-25-design-gap-06-07-4063/README.md) |
| GAP-08 | Verified the active underline, primary and muted tab ink, authored asset swatches, and native close. The dock search appears when another panel is active. [Asset tabs](receipts/2026-11-25-design-gap-08-15-4063/README.md) · [Dock search](receipts/2026-11-25-design-gap-08-14-15-4063/README.md) |
| GAP-09 | Verified the new command artwork in Manage, View, Assemble, and Place Views. All 245 catalog icons rasterized in the three themes. Drawing artwork uses the themed drawing glyphs. [Native and catalog checks](receipts/2026-11-25-design-gap-07-09-11-4063/README.md) |
| GAP-10 | Verified the centered document and model-state title, flush 46 × 32 px window controls, and rose Close hover. The brand mark uses the 16 px handoff size. [Native frame](receipts/2026-11-25-design-gap-10-4063/Titlebar.png) |
| GAP-11 | Added group separators, centered uppercase captions, and the compact density icon and menu. Verified the compact Create keyboard route and the disclosure control with captions hidden. [Verification](receipts/2026-11-25-design-gap-07-09-11-4063/README.md) |
| GAP-12 | Verified the taller icon-led document strip and its context menu. Native close and reorder workflows use the same tab engine. [Native checks](receipts/2026-11-25-design-gap-12-13-4063/README.md) |
| GAP-13 | Verified native pointer coordinates, document units, and camera zoom. The status bar also retains the sketch dimension message and displays DOF. [Native checks](receipts/2026-11-25-design-gap-12-13-4063/README.md) |
| GAP-14 | Added document, body, feature, sketch, and origin icons, active representation checks, mono DOF, and the dashed rollback rule. Verified inline clear and query focus. [Native checks](receipts/2026-11-25-design-gap-08-14-15-4063/README.md) |
| GAP-15 | Verified ViewCube contrast, neutral drawing paper, and Part asset panels leaving Drawing. Model selection uses Studio teal. Dimensions use mono units, collision spacing, and extension lines. [View and Drawing](receipts/2026-11-25-design-gap-08-15-4063/README.md) · [Geometry checks](receipts/2026-11-25-design-gap-08-14-15-4063/README.md) |
| GAP-16 | Verified that the first frame with the View commands underlines View and clears the previous 3D Model underline. The regression reads rendered text and the underline quad from the same frame. [GUI test](../../crates/gui/tests/ribbon_click/tab_underline.rs) |
| GAP-17 | New profiles use 260 px side docks when space allows. The central area keeps at least 55% of available width, and widening restores the preferred dock sizes. Recent hides lower-priority columns to protect filenames. The strip reveals the active tab on resize and shortens long names in the middle. [Resize evidence](receipts/2026-11-25-design-gap-17-18-20-4063/README.md) |
| GAP-18 | The narrow 3D Model ribbon uses compact icon-and-caption buttons and keeps its last group reachable at 780 px. Floating modeling properties use the live viewport bounds, and native Extrude and Chamfer checks cover a resize to 980 × 540 and back. The compact Sketch row still needs the short-window check. [Ribbon resize](receipts/2026-11-25-design-gap-17-18-20-4063/README.md) · [Property resize](receipts/2026-11-25-design-gap-18-21-4063/README.md) |
| GAP-19 | The sizing and alignment of the ribbon tab row, additional command flyouts, and compact Sketch disclosure remain open. The dock, document strip, and 3D Model compaction changes are covered under GAP-12, GAP-17, and GAP-18. |
| GAP-20 | Home in front of an open Part shows the File menu and Home actions without the Part ribbon, Model, or asset docks. Returning restores the Part layout and sketch editing context. The published no-document startup workspace keeps its empty browsers. [Home evidence](receipts/2026-11-25-design-gap-17-18-20-4063/README.md) |
| GAP-21 | Light uses a neutral disabled OK button with slate text. Read-only Theme Editor values use dark text and bordered swatches, the editor has a floating shadow, and New name is placed before its field. The Light ViewCube has dark label ink when fully opaque. Its default inactive fade still reduces label contrast. [Native checks](receipts/2026-11-25-design-gap-18-21-4063/README.md) |
