# Edit and save project settings

Edit Settings opens a draft for the inspected project. Save writes the
name, workspace, templates, design data, and reference library folders to
the existing project file. The desktop validates the paths and project
assets before updating the loaded configuration. A disk check compares
the current configuration with the loaded one before atomic replacement.

The native regression cancelled a name edit and compared the original
file bytes. It then edited every supported field and changed the file
outside the application. Save left the external content and loaded project
intact. After restoring the baseline, retrying saved every edited field
and refreshed the chooser while the open Part remained active.
[Test results](test-results.txt)

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect project information from the template chooser | [Report](wayland/inspect.json) | [Report](x11/inspect.json) |
| Inspect an inactive project without activating it | [Report](wayland/inactive.json) | [Report](x11/inactive.json) |

The build used `d32cc0861` with the project editor and existing local
filesystem and marking-menu edits. Wayland used headless Weston. X11 used
Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256) · [Bundled template input hash](bundled-input.sha256)
