# Place Sketch Text with a mouse drag

Two native tests passed on Linux with X11, Xvfb, and Vulkan Lavapipe.
The drag test moved the pointer from `[1050, 710]` to `[1250, 810]`
in the test's 1440 × 900 window. The empty Format Text dialog appeared
on release. Accepted text used the sketch coordinates of the initial
press. The Escape test canceled an active press and left no text draft
or accepted text after release.

[Format Text after the native drag](format-text-after-drag.png)

All 16 steps of point-placement scenarios `text-02` and `text-07`
also passed. They cover accepting an empty editor, keeping the next
placement available, and restoring accepted text with Undo and Redo.
The scenario file at revision `b576445b98cffd686dcc05aa8c017de2a591c8c8`
had SHA-256 `0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

This record covers drag placement. Pinned `text-16` still needs the
portable Part fixture and an editable field linked to parameter `d0`.
