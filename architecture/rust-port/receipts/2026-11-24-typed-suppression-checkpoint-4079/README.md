# Typed feature suppression checkpoint replay

Issue #4079. The owner regression opens a filleted Part and retains its
current replay checkpoints. The typed command suppresses and restores the
Fillet. Before the change, the first typed suppression advanced the document
from generation 2 to 3 while its published result remained at generation 2.
After the change, each reply reports the suppression state and an empty
health reason, and the current result already reflects the edit. The
checkpoint remains available for the next edit.

The command projects the feature status from the candidate replay when the
certified backend opts in and the document has a current checkpoint under
the same policy. A second regression retains a checkpoint under a different
policy and checks that the suppression command leaves it stale. Edits without
a safe prefix retain the backend's full health check. Injected backends retain
their own validation by default.

| Pinned Model Browser scenario | Before | Current |
| --- | --- | --- |
| Switch between states with different feature suppression | [11 steps](before/switch/cucumber.json) | [11 steps](pinned/switch/cucumber.json) |
| Cancel a staged model-state suppression change | [13 steps](before/cancel/cucumber.json) | [13 steps](pinned/cancel/cucumber.json) |

The native runs use `b576445b98cffd686dcc05aa8c017de2a591c8c8` on
Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe.
