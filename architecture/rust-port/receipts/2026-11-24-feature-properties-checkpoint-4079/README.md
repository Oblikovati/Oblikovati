# Feature Properties checkpoint replay

Issue #4079. An owner test opens a real filleted prism and retains its
current feature checkpoints. It applies the same command as the Feature
Properties dialog, which renames the Fillet and suppresses it when the
width exceeds 40 mm. Before the change, the command left a result for
generation 2 after the document advanced to generation 3.

The edited result now has a current checkpoint, a suppressed Fillet,
and the same prism face names and areas as an independent full replay.
Undo restores the Fillet and redo restores the suppression.

A native regression accepts Always on an extrusion in Feature
Properties and checks that the published scene becomes empty. The
native Undo shortcut restores the solid.

| Model Browser scenario | Published before | Published current | Pinned current |
| --- | --- | --- | --- |
| Inspect and cancels conditional feature suppression | [17 steps](before/controls/cucumber.json) | [17 steps](published/controls/cucumber.json) | [16 steps](pinned/controls/cucumber.json) |
| Switch between unconditional and conditional suppression | [15 steps](before/modes/cucumber.json) | [15 steps](published/modes/cucumber.json) | [14 steps](pinned/modes/cucumber.json) |

The published source is `f590a2cc540119948799f5733a5fc2276e6758cc`.
The pinned source is `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
Runs use Linux X11, Xvfb, Openbox, and Vulkan Lavapipe.
