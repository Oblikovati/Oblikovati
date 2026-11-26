# Docking System on Wayland

All 13 published scenarios and 134 steps passed in isolated headless Weston
sessions with Native PartOne and the saved solid Part. Coverage includes
edge previews, splitter resize, joining and restoring tabs, floating and
redocking, switching documents, and restoring a saved layout across processes.
[Combined report](cucumber.json) · [Published input hashes](inputs.sha256)

The mixed-dock document-switch scenario originally stopped when its
driver tried to send an X11 click. It now uses the same backend-aware
pointer driver as the dock tab menu. It checks both dock regions and verifies
that the viewport changes to the empty Part and back to the shaded solid.
[Wayland baseline](wayland-retention-baseline.json) · [X11 rerun](x11-retention.xml)

The restart scenario saves the arrangement created by the test gestures and
launches two standalone desktop processes with one temporary profile. Each
Wayland process renders 20 frames and saves its layout on exit. The verifier
compares the separate Model dock and both browser tabs against the original
positions, sizes, dock IDs, and tab order. The native scenario also verifies
the visible solid when reopening. The X11 driver compares screen captures
across the restart.
[Wayland dock records](wayland-restart.json) · [X11 restart report](x11-restart.xml)

The build used `9f67524cf` and existing local GUI filesystem edits. The dock
menu, mixed-dock switching, and restart cases were rerun with the verifier
changes. Wayland used scripted pointer events, Weston with Pixman, and a
1440 × 900 output. X11 used XTest, Xvfb, and Openbox. Both used Vulkan Lavapipe.
Issue #3978 tracks strict build and runtime provenance.
