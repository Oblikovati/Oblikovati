# Pick a Fillet edge after starting the tool

The native-window test starts Fillet from the ribbon with no edge selected,
checks that an empty draft cannot commit, clicks a solid edge, observes a
preview, and confirms the feature. The run used scripted native-window mouse
and keyboard events under Xvfb with Vulkan Lavapipe. It closed the window.

`preview.png` was inspected. The panel identifies one selected edge and shows
the preview. The replacement preview hides the original edge highlight, so the
generic scenario's distinct selected-state requirement remains unverified.
The live scenario also requires fixture and capture evidence. This test is a
focused diagnostic, not full BDD acceptance or a frame-time measurement.
