# Pinned ribbon docking and visibility coverage

The native benchmark ran the three scenarios against the clean pinned source
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. It used Linux X11 under Xvfb with
Vulkan Lavapipe.

| Scenario | Native steps |
| --- | ---: |
| [Finish a sketch while the ribbon is docked left](finish-sketch-cucumber.json) | 9 |
| [Keep the ribbon docked while switching document types](document-switch-cucumber.json) | 11 |
| [List 3D Model panels from the side-docked tab menu](panel-inventory-cucumber.json) | 8 |

All 28 steps passed. The same three scenarios also passed against the current
shared scenario source with its named CAD fixtures. The panel inventory run
provides an R0 record for the pinned source.
