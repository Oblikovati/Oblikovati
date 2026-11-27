# Model state and view controls in an empty Part

Six pinned native scenarios passed all 88 steps after the empty-viewport
check was updated to use the document viewport bounds.

| Scenario | Passed steps | Receipt |
| --- | ---: | --- |
| Toggle the model-state edit scope | 11 | [Run](edit-scope/cucumber.json) |
| Duplicate the active model state in an empty part | 16 | [Run](duplicate-state/cucumber.json) |
| Copy and activates a view representation in an empty part | 15 | [Run](copy-view/cucumber.json) |
| Toggle camera autosave for an inactive copied view | 17 | [Run](autosave/cucumber.json) |
| Choose and restore annotation scale for an active view | 17 | [Run](scale/cucumber.json) |
| Restore a saved camera for an active view representation | 12 | [Run](camera/cucumber.json) |

The runs used `7de9a444a` with the existing local file-browser edits on
Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The pinned source commit
was `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The six scenarios gained R0
baseline records. See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
