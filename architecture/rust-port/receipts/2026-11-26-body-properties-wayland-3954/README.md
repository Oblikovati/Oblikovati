# Current Body Properties on Wayland and X11

All six published scenarios and 69 steps passed on each Linux backend.
Update calculated all six physical fields and kept their values when
the dialog reopened. Applying Black changed the sampled solid face,
and selecting As Part restored its original appearance. Cancel, Escape,
and the title-bar close control discarded pending appearance changes.

The original Update step found the toolbar command while the dialog
was open. The revised step targets the dialog's trailing Update caption
even when the text reader joins it with the body name. The test records
a solid-face sample before editing so sketch annotations cannot satisfy
the appearance check.

The build used `a8b8179aa` with the [test source changes](test-input.sha256),
the already verified Quick Access test changes, and the existing local
GUI filesystem test-module visibility edit. Wayland used nested Weston
13 and X11 used Openbox. Both used Xvfb and Vulkan Lavapipe. The
[published input hashes](current-inputs.sha256) matched before and after
both runs. Strict build and runtime verification remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
