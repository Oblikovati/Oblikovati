# Pinned marking-menu selector and cancellation coverage

The native benchmark used clean MCAD-Scenarios revision
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 under Openbox
and Xvfb with Vulkan Lavapipe. The application used `9b3267b94` with
the test setup change committed as `162cead51`.

| Scenario | Native steps |
| --- | ---: |
| [Inspect marking-menu sub-environment choices without changing selection](sub-environments.json) | 9 |
| [Inspect marking-menu environments without changing selection](environments.json) | 8 |
| [Inspect marking-menu presentation choices without changing settings](presentation.json) | 12 |
| [Cancel a staged classic context-menu setting](classic-cancel.json) | 11 |
| [View 3D Sketch marking-menu locations and sub-environments](3d-sketch.json) | 12 |

All 52 steps passed. The Go marking-menu editor test cited in the
classifications establishes the reference editor rendering by source review.
