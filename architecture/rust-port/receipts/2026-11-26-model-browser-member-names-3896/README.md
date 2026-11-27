# Model State member names on Wayland and X11

All three published scenarios passed, with 44 steps on each backend. The
table highlighted both duplicate names, offered Yes and No, and preserved
the original names when the user declined and cancelled. A valid rename
updated the browser while Copy State stayed active. The empty-name warning
closed the table on cancellation and preserved the committed name.

A separate native test accepted duplicate removal and verified that one
member with the existing name remained.

- Duplicate names on [Wayland](duplicate-wayland.json) and [X11](duplicate-x11.json)
- Rename on [Wayland](rename-wayland.json) and [X11](rename-x11.json)
- Empty name on [Wayland](empty-wayland.json) and [X11](empty-x11.json)
- Accepted duplicate removal on [Wayland](accept-duplicate-wayland.txt) and [X11](accept-duplicate-x11.txt)

The build used `e422388c9` and the existing local GUI filesystem
test-module visibility edit. Wayland used nested Weston 13, X11 used
Openbox, and both used Xvfb with Vulkan Lavapipe. The
[published input hashes](current-inputs.sha256) matched before and after
the run. Strict build and runtime verification remains open under #3978.
