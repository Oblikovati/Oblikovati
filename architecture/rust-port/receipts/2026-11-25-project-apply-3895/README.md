# Activate a project from Projects

Apply makes the inspected project active for the current session. The desktop
keeps the active document and reloads the active project's templates. The
chooser drops the old selection and shows an available folder in the new
catalog. Single-click inspection retains the original active project.

The native regression selected a template in the original project's Metric
catalog, inspected an alternate project, and applied it. It checked the active
project mark, the unchanged document identity, the alternate project's
English-only catalog, and the empty template selection after closing Projects.
[Test results](test-results.txt)

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect project information from the template chooser | [Report](wayland/inspect.json) | [Report](x11/inspect.json) |
| Inspect an inactive project without activating it | [Report](wayland/inactive.json) | [Report](x11/inactive.json) |

The build used `9df6f768d` with the activation changes and existing local
filesystem and marking-menu edits. Wayland used headless Weston. X11 used
Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256) · [Bundled template input hash](bundled-input.sha256)
