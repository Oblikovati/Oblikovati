# Pinned Line Close and Restart coverage

The native benchmark used the clean scenario revision
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and Linux X11 under Xvfb and
Openbox with Vulkan Lavapipe. The pinned step bindings are in `4eb81101f`.

| Scenario | Native steps passed |
| --- | ---: |
| [Close a two-segment chain and inspect loop recognition](close-cucumber.json) | 8 |
| [Restart without removing an accepted segment](restart-cucumber.json) | 8 |
