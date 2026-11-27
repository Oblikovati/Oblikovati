# Restore a template when the startup project loads after Home

The published scenario loads the Generic template project, selects its
Metric Standard (mm) Part, cancels New document, and reopens the application
with the same profile. It checks the restored selection, units and material,
Create and Cancel availability, and the existing Part and Sketch1 after Escape.

On the [Wayland baseline](baseline.json), the chooser showed the Metric
folder after restart and had no selected template. The first Home frame had
no project, so loading the startup project cleared the path read from the
profile. The chooser now keeps that path through the initial load and restores
the selection from the matching catalog entry. A focused GUI test also checks
that switching to a different loaded project clears the remembered path.

All 22 scenario steps passed on both backends and all 63 focused GUI tests
passed. [Wayland report](wayland.json) · [X11 report](x11.xml) ·
[Published input hashes](inputs.sha256)

The build used `fa3deaad9` with this change and existing local GUI filesystem
edits. Wayland used headless Weston with Pixman. X11 used Xvfb and Openbox.
Both rendered with Vulkan Lavapipe. Issue #3978 tracks strict provenance.
