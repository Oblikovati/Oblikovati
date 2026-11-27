# Document Opening with the current published fixtures

Both desktop backends passed all 27 Document Opening scenarios and 315
steps. The runs cover the saved assembly opened from Home, drawing sheets
and base views, deferred updates, reopening an existing document in its
original tab, selection and navigation, properties and previews, and the
file context menu.

The full runs used `70b8fd5cc`. The context-menu reruns verified Open,
Copy, Rename, and Properties within one popup draw list. Each combined
report contains that backend's rerun. Separate native pointer tests
right-clicked the thumbnail and dismissed the menu in the file-list
gutter on both desktops. The dialog retained the name and preview and
cleared the selection.

The head library runner used `OBK_BDD_AREA=application`,
`OBK_BDD_FEATURE=document-opening`, and the full shared scenario directory
as `OBK_BDD_ROOT`. Wayland used nested Weston 13 and X11 used Openbox.
Both used Xvfb and Vulkan Lavapipe. The feature sources stayed stable
during execution. Published fixture hashes and the scenario support files
are recorded with the reports. Issue #3978 tracks strict provenance.

- [Wayland suite](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 suite](x11.json) and [step arguments](x11-arguments.json)
- Context menu reruns on [Wayland](wayland-context-menu.json) and [X11](x11-context-menu.json)
- [Current input hashes](current-inputs.sha256)
