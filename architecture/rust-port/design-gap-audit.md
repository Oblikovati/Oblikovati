# Desktop design gap audit

Source: `/mnt/vininas-projects/MCAD-Scenarios/Design/DESIGN-GAP-AUDIT.md`.
The designer audited a Windows binary on 2026-11-25. Native evidence recorded
here uses the Rust desktop on Linux under Xvfb, Openbox, and Vulkan Lavapipe.

## Verification

| Gap | Evidence or remaining work |
| --- | --- |
| GAP-01 | Native Light Mode switch, theme-aware Default backdrop, explicit High Contrast override, and a light status bar. [Default](receipts/2026-11-25-design-gap-01-4063/Light.png) · [Override](receipts/2026-11-25-design-gap-01-4063/Override.png) |
| GAP-02 | Verify that the scrollable properties content leaves the action footer visible and that a new feature draft replaces the old one. |
| GAP-03 | Verify Sketch ribbon compaction, row clearance, compact group labels, and Finish Sketch styling. |
| GAP-04 | Verify translucent, seamless origin planes beneath viewport controls. |
| GAP-05 | Bring the properties controls, placement, and floating elevation into alignment. |
| GAP-06 | Verify File menu iconography, row heights, long names, and pin controls. |
| GAP-07 | Verify Home recent-document columns, selection, and action hierarchy. |
| GAP-08 | Verify dock tab selection, header controls, and asset swatches. |
| GAP-09 | Existing artwork is mapped for several ribbons. Verify the remaining command mappings. [Ribbon icon checks](receipts/2026-11-25-identity-ribbon-icons-4063/README.md) |
| GAP-10 | Audit the existing brand mark and window controls, and verify title alignment. |
| GAP-11 | Verify ribbon group separators, captions, and density control. |
| GAP-12 | Verify document tab spacing, artwork, and close controls. |
| GAP-13 | Verify viewport telemetry in the status bar. |
| GAP-14 | Verify browser row artwork, representation state, rollback marker, and search. |
| GAP-15 | Verify ViewCube face labels, dimensions, selection colors, and Drawing workspace styling. |
| GAP-16 | Verify that an input event updates the active tab underline and body together. |
