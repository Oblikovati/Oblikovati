# Pinned tabs-only ribbon Finish coverage

The native benchmark used the clean scenario revision
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and Linux X11 under Xvfb and
Openbox with Vulkan Lavapipe. The Finish step captures the active document
identity before clicking the command and verifies that the same Part returns.
The test setup change is in `c230ced7b`.

| Scenario | Native steps passed |
| --- | ---: |
| [Change ribbon context while editing an existing sketch](finish-tools-cucumber.json) | 11 |
| [Finish a sketch from a tabs-only ribbon](finish-sketch-cucumber.json) | 8 |
