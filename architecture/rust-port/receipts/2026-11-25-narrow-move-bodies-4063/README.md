# Move Bodies properties at the minimum window size

At 640 × 480, the old Move Bodies dialog put its labels after the controls
and left about 100 pixels below Cancel. The dialog now aligns the body
selector and X, Y and Z entries in a shared value column, outlines their
edges, and fits its content and action footer inside the viewport.
[Before](before.png) · [After](after.png)

The native Modify flyout regression failed on the old label position.
It now verifies the left label column, viewport containment and footer
clearance. Clicking Cancel with X11 input closes the dialog without changing
the feature count. The eight native ribbon keyboard tests, six GUI tests
and three owner tests for Move Bodies also passed.

Two published Ribbon Panel Expansion scenarios passed all 19 steps.
They launch Move Bodies via keyboard and from the pinned expansion, then
cancel without applying the move.
[Keyboard launch](keyboard-launch.json) · [Pinned Cancel](pinned-cancel.json)
· [Scenario and fixture hashes](current-inputs.sha256)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`c9ab85e54` with the Move Bodies layout change and pending local filesystem
and Ribbon Customization edits.
