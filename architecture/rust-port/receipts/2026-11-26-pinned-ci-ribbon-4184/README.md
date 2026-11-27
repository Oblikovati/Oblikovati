# Pinned ribbon customization coverage

The native benchmark opened each scenario from the clean pinned source
`b576445b98cffd686dcc05aa8c017de2a591c8c8`, using the application code in
`ee90cd41c5b7cf1efce24c3f18673b7962d1f72a`. The run used Linux X11 under
Xvfb with Vulkan Lavapipe.

| Scenario | Native steps |
| --- | ---: |
| [Inspect marking-menu customization for a part environment](marking-cucumber.json) | 16 |
| [Inspect custom panel targets from a part document](target-cucumber.json) | 12 |
| [Retain command sort order across category switches until the dialog closes](sort-cucumber.json) | 11 |
| [Resize ribbon customization while a command is staged](resize-cucumber.json) | 14 |
| [Reopen ribbon customization at its default size after resizing](reopen-cucumber.json) | 7 |

The five runs provide R0 baseline evidence for the pinned scenario source.
All 60 steps passed in native windows. The Go registry and editor tests cited
in the classifications establish reference behavior by source review.
