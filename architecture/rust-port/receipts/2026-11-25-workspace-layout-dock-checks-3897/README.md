# Native Workspace Layout dock checks

Five current cases passed 46 steps on Linux X11 with Xvfb, Openbox and
Vulkan Lavapipe. The published Part opens with a right-side asset dock.
The boundary checks verify that the viewport reaches the actual ImGui
splitter and that the visible dock reaches the application edge. The
document strip follows the viewport and the status row spans the frame.

Restoring the Model Browser now checks its active dock tab after the
follow-up native frames. The Drawing case then verifies its resources and
sheet tree. The Part case verifies that the Origin tree and the active
sketch are both visible.

The test build used `2f7418cf6`, the changes in this commit, and existing
local GUI filesystem, opening and saving edits. The
[input hashes](current-inputs.sha256) matched the isolated source and
fixtures. Head test Clippy passed with warnings denied. Issue #3978
tracks strict build and runtime provenance.
