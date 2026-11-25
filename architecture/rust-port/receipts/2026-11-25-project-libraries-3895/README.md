# Carry libraries into a new project

Select Libraries lists the bound reference library roots of registered
projects. The transfer controls add and remove roots. Going back to the
workspace page preserves the chosen order. Finish writes each chosen root
at its resolved absolute location so a new workspace can use a library
that was recorded relative to another project.

The native regression registered a project with two relative library roots
and a duplicate spelling. It transferred both, removed the first, navigated
back and forward, and added the first again. Decoding and binding the saved
project confirmed the resulting search order and the new workspace.
[Test results](test-results.txt)

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect and cancel the new single-user project wizard | [Report](wayland/cancel.json) | [Report](x11/cancel.json) |
| Navigate backward through the new-project wizard | [Report](wayland/back.json) | [Report](x11/back.json) |

The build used `9e090f650` with the library changes and existing local
filesystem and marking-menu edits. Wayland used headless Weston. X11 used
Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256)
