# Model Browser model states and view visibility

The full Wayland diagnostic run on `96b2086bb` passed 74 of 100 scenarios
and 985 steps in passing scenarios. Its 26 failures include 12 cases
using X11-only test input and four that require a valid extrusion taper.
The [full report](full-wayland.json) and [resolved arguments](full-wayland-arguments.json)
record the run before the focused fixes.

Seven focused cases passed 97 steps on each Linux backend. Model-state
checks at `d495226d4` verify that the branch shows the active member, that
duplication keeps the source active until the duplicate is selected, and
that toggling scope updates the badge, tooltip, and browser root.

- Create and switch on [Wayland](model-state-branch-create-wayland.json) and [X11](model-state-branch-create-x11.json)
- Duplicate on [Wayland](model-state-branch-duplicate-wayland.json) and [X11](model-state-branch-duplicate-x11.json)
- Edit scope on [Wayland](model-state-branch-scope-wayland.json) and [X11](model-state-branch-scope-x11.json)

Visibility checks at `3dce942bc` verify the published visible-body count
and a rendered point sampled from the solid before hiding it. Saved
sketch annotations stay visible. The tests verify that locked and unlocked
copies restore their expected visibility and that bulk visibility commands
leave the origin plane alone in an empty Part.

- Hide and show on [Wayland](model-view-visibility-wayland.json) and [X11](model-view-visibility-x11.json)
- Locked view on [Wayland](model-view-locked-wayland.json) and [X11](model-view-locked-x11.json)
- Unlocked view on [Wayland](model-view-unlocked-wayland.json) and [X11](model-view-unlocked-x11.json)
- Empty part on [Wayland](model-view-empty-wayland.json) and [X11](model-view-empty-x11.json)

The builds included the existing local GUI filesystem test-module visibility
edit. Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source before and after the runs. Strict verification remains open under #3978.
