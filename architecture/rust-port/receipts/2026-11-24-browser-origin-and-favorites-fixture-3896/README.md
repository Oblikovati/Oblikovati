# Origin navigation and Favorites on the saved Part

The published scenarios now keep the named workspace Part active when
the setup calls for a visible block. The setup checks the active
document, its extrusion and visible body, and records the initial
camera for the view assertions.

| Native scenario | Pinned | Published |
| --- | --- | --- |
| Navigate the origin branch with arrow keys | [19 steps passed](pinned/navigate/cucumber.json) | [20 steps passed](published/navigate/cucumber.json) |
| Move between origin planes with Shift and Down Arrow | [11 steps passed](pinned/shift/cucumber.json) | [12 steps passed](published/shift/cucumber.json) |
| Change the Favorites pane view mode | [14 steps passed](pinned/favorites/cucumber.json) | [15 steps passed](published/favorites/cucumber.json) |

The runs used `ee5ec8fbe` with the local setup change and existing
file-browser edits on Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe.
The pinned source was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The [published manifest](../2026-11-24-model-browser-named-part-3896/snapshot.json)
records the named fixture. Both pinned origin scenarios gained R0 records.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
