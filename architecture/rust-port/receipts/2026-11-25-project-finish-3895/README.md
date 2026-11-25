# Finish a single-user project

Finish on the last wizard page submits the name and workspace to the
desktop owner. The owner serializes the schema-1 project and publishes
it at the derived path. The project records its workspace relative to
the file. The new row is available for inspection as soon as the wizard
receives the successful result.

The native regression occupied the default project path with known bytes.
It confirmed that Finish left those bytes and the project list intact and
displayed the failure in the wizard. After renaming the project, the same
wizard saved successfully. The existing project stayed active, and a fresh
desktop owner could open the saved file. The storage tests independently
verified that atomic project creation preserves the existing file and leaves
no temporary file after a collision. The GUI tests checked that the wizard
accepts its own result and can retry after a failure.
[Test results](test-results.txt)

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect and cancel the new single-user project wizard | [Report](wayland/cancel.json) | [Report](x11/cancel.json) |
| Navigate backward through the new-project wizard | [Report](wayland/back.json) | [Report](x11/back.json) |

The build used `164d3b070` with the creation changes and existing local
filesystem and marking-menu edits. The Wayland cases used headless Weston.
X11 used Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256)
