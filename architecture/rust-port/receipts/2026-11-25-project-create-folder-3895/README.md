# Create a workspace folder in the project wizard

Make New Folder opens a name editor for the current folder. Create adds one
directory and selects it in the browser. OK accepts that selection as the
workspace and updates the derived project file. Cancelling the name editor
creates nothing. Cancelling the browser after Create keeps the new directory
and the wizard's previous workspace.

The native regression used a temporary Documents directory. It cancelled a
name, created a different directory, cancelled the browser, reopened it,
and accepted the new directory. It checked the filesystem and the previewed
project file path. The GUI tests also checked invalid names and confirmed
that a duplicate name leaves the existing directory and its contents intact.
[Test results](test-results.txt)

Both published workspace-folder scenarios passed on both backends.

| Scenario | Steps | Wayland | X11 |
| --- | ---: | --- | --- |
| Cancel browsing for a project workspace folder | 14 | [Report](wayland/cancel.json) | [Report](x11/cancel.json) |
| Choose the workspace folder for a new project | 11 | [Report](wayland/choose.json) | [Report](x11/choose.json) |

The build used `32bae902f` plus the folder creation changes and pre-existing
local filesystem and marking-menu edits. Wayland used headless Weston.
X11 used Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256)
