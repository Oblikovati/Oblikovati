# Saved document tab preview on a network path

All 41 amended Document Tabs scenarios passed on Wayland and X11,
with 352 steps on each backend. The new case opened the published Part
directly from a mounted CIFS share. It verified a textured thumbnail
above the tab, the user-facing saved path, and a close mark that
appeared on hover without shifting the adjacent tab.

The tooltip uses a filtered texture up to 384 × 216 pixels. Saved
document thumbnails keep their existing dimensions. The 2× placement
test passed on both backends. A separate native Linux test injected
Windows UNC metadata and verified its display while the actual saved
path stayed intact. The GUI tests also covered extended drive paths,
averaging at downsampled edges, and retained frame texture ownership.
A native Windows run remains unverified.

The build used the preview changes in `76eb93957` and existing local
GUI filesystem edits. The full runs preceded moving the frame capture
call into the resources module. The CIFS scenario was rerun on Wayland
after the move. Wayland used nested Weston 13 and X11 used Openbox, with
Xvfb and Vulkan Lavapipe on both. The published input hashes matched
before and after the full runs. Issue #3978 tracks strict provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
