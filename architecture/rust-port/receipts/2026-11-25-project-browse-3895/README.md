# Browse for an existing project

Browse opens a project-only file picker in the inspected project's folder,
or in the default folder when no project is available. The desktop uses
the same project loader as startup. A load error keeps the picker open.
Success closes it and shows the registered project's settings. An already
active project remains active.

The native regression opened the picker over the template chooser, verified
the project-only filter, refused an unsupported schema, and cancelled after
selecting a valid file. It then opened that file and checked that its settings
were displayed. Opening the same file by its absolute path retained one
registration. The Finish regression passed through the same owner action.
The project window redraws after success so the picker title disappears
and the new settings render without another input event.
[Test results](test-results.txt)

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect project information from the template chooser | [Report](wayland/inspect.json) | [Report](x11/inspect.json) |
| Inspect an inactive project without activating it | [Report](wayland/inactive.json) | [Report](x11/inactive.json) |

The build used `c47d063be` with the Browse changes and existing local
filesystem and marking-menu edits. Wayland used headless Weston. X11 used
Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256)
