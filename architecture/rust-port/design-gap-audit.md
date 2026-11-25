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
| GAP-07 | Verified the filename and metadata columns, distinct type icons, active List and Grid styles, visible checkboxes, and primary action. The Home rail shows the brand mark and the active project when available. [Native checks](receipts/2026-11-25-design-gap-06-07-4063/README.md) |
| GAP-08 | Verify dock tab selection, header controls, and asset swatches. |
| GAP-09 | Existing artwork is mapped for several ribbons. Verify the remaining command mappings. [Ribbon icon checks](receipts/2026-11-25-identity-ribbon-icons-4063/README.md) |
| GAP-10 | Verified the centered document and model-state title, flush 46 × 32 px window controls, and rose Close hover. The brand mark uses the 16 px handoff size. [Native frame](receipts/2026-11-25-design-gap-10-4063/Titlebar.png) |
| GAP-11 | Verify ribbon group separators, captions, and density control. |
| GAP-12 | Verify document tab spacing, artwork, and close controls. |
| GAP-13 | Verify viewport telemetry in the status bar. |
| GAP-14 | Verify browser row artwork, representation state, rollback marker, and search. |
| GAP-15 | Verify ViewCube face labels, dimensions, selection colors, and Drawing workspace styling. |
| GAP-16 | Verified that the first frame with the View commands underlines View and clears the previous 3D Model underline. The regression reads rendered text and the underline quad from the same frame. [GUI test](../../crates/gui/tests/ribbon_click/tab_underline.rs) |
