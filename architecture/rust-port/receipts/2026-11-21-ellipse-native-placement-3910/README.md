# Ellipse placement with the native YZ Part

Five focused published scenarios passed with cleanup under Xvfb and Vulkan
Lavapipe. The runner sends scripted input to the desktop window. It opens a
writable copy of the supplied native Part, edits Sketch1 in RIGHT view, and
projects the published millimetre coordinates through the active camera. The
scenario steps cover three-point placement, cancel after
the center or first axis, Undo and Redo, and rejection of a coincident first
axis without advancing. The screenshot shows the rotated ellipse after Redo.

The published feature hash is recorded with the native fixture hash. Later
generic Given steps retain the open Part and the active placement in ellipse-03.
The five corresponding pinned scenarios also passed. The full Ellipse feature
requires further verification.

The application build used the changes in the commit that adds this receipt
on top of `cd16c1576`. Native evidence still needs sealed build and runtime
inputs for strict acceptance. See
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
