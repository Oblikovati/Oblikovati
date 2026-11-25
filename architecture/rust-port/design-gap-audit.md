# Desktop design gap audit

Source: `/mnt/vininas-projects/MCAD-Scenarios/Design/DESIGN-GAP-AUDIT.md`.
The designer audited a Windows binary on 2026-11-25. Native evidence recorded
here uses the Rust desktop on Linux under Xvfb, Openbox, and Vulkan Lavapipe.

## Verification

| Gap | Evidence or remaining work |
| --- | --- |
| GAP-01 | Native Light Mode switch, theme-aware Default backdrop, explicit High Contrast override, and a light status bar. [Default](receipts/2026-11-25-design-gap-01-4063/Light.png) · [Override](receipts/2026-11-25-design-gap-01-4063/Override.png) |
| GAP-02 | Verified the pinned Extrude footer and exclusive feature drafts. Content-based floating height and the Apply & Create New affordance remain open. [Native check](receipts/2026-11-25-design-gap-p0-4063/README.md) |
| GAP-03 | Verified labelled Sketch groups, last-row clearance, and the emerald Finish Sketch at 1440 px. [Native check](receipts/2026-11-25-design-gap-p0-4063/README.md) |
| GAP-04 | Verified translucent origin-plane selection beneath the viewport controls, with the ViewCube kept clear and plane selection still working. [Native check](receipts/2026-11-25-design-gap-p0-4063/README.md) |
| GAP-05 | Verified 40/60 Fillet and Chamfer fields, upper-left placement, primary OK, inline Distance A tools, and a single direction row. The profile picker, preview toggle, and floating shadow remain open. [Native checks](receipts/2026-11-25-design-gap-05-4063/README.md) |
| GAP-06 | Verified the wider File menu, command and document icons, distinct pin controls, and long-name truncation. All eight published scenarios passed. [Native checks](receipts/2026-11-25-design-gap-06-07-4063/README.md) |
| GAP-07 | Verified the filename and metadata columns, distinct type icons, active List and Grid styles, visible checkboxes, and primary action. The Home rail uses the themed wordmark and shows the active project when available. [Native checks](receipts/2026-11-25-design-gap-06-07-4063/README.md) |
| GAP-08 | Verified the active dock underline and titlebar close action. The asset choices display swatches from the authored appearance colors. Model Browser header duplication and active text weight remain open. [Native checks](receipts/2026-11-25-design-gap-08-15-4063/README.md) |
| GAP-09 | Verified the new command artwork in Manage, View, Assemble, and Place Views. All 245 catalog icons rasterized in the three themes. Drawing artwork uses the themed drawing glyphs. [Native and catalog checks](receipts/2026-11-25-design-gap-07-09-11-4063/README.md) |
| GAP-10 | Verified the centered document and model-state title, flush 46 × 32 px window controls, and rose Close hover. The brand mark uses the 16 px handoff size. [Native frame](receipts/2026-11-25-design-gap-10-4063/Titlebar.png) |
| GAP-11 | Added group separators, centered uppercase captions, and the compact density icon and menu. Verified the compact Create keyboard route. [Verification](receipts/2026-11-25-design-gap-07-09-11-4063/README.md) |
| GAP-12 | Verified the taller icon-led document strip and its context menu. Native close and reorder workflows use the same tab engine. [Native checks](receipts/2026-11-25-design-gap-12-13-4063/README.md) |
| GAP-13 | Verified native pointer coordinates, document units, and camera zoom. The status bar also retains the sketch dimension message and displays DOF. [Native checks](receipts/2026-11-25-design-gap-12-13-4063/README.md) |
| GAP-14 | Verify browser row artwork, representation state, rollback marker, and search. |
| GAP-15 | Verified Light and Dark ViewCube contrast, neutral drawing paper, and hiding Part assets in Drawing. Dimension presentation and model selection color remain open. [Native checks](receipts/2026-11-25-design-gap-08-15-4063/README.md) |
| GAP-16 | Verified that the first frame with the View commands underlines View and clears the previous 3D Model underline. The regression reads rendered text and the underline quad from the same frame. [GUI test](../../crates/gui/tests/ribbon_click/tab_underline.rs) |
