# Coincident from the compact Sketch ribbon

Issue #3857. At `60fdab214`, the pinned standalone-point scenario stopped at
its first Coincident command. At the native harness's default window width,
Constrain is a compact ribbon panel and its command captions appear only when
the panel opens. The driver now opens the rendered compact control before
selecting Coincident. Wide layouts still use the visible command directly.

On Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe, the pinned
**Constrain two standalone points and inspect history** case passed all 12
steps, including the second-point prompt, the coincidence marker, undo and
redo. See the [Cucumber result](pinned-point-history/cucumber.json) and
[source revision](pinned-point-history/source-revision.json). The feature's
SHA-256 before and after was
`a2d31789402dfca6ebb083bc02002b722f8518266c822d8c04b4ae45b8251c7f`.
The pinned [oblique segment](pinned-segments/cucumber.json) and
[point context menu](pinned-context/cucumber.json) cases each passed all
10 steps after binding their current wording to the existing native steps.

## Parameter-driven cube

Both cube tests initially left two translation degrees of freedom. The
default Geometry in current command scope excludes the projected origin.
The X11 journey now places the rectangle away from the origin and applies
Coincident through the ribbon. It passes the zero-DOF and size-expression
checks, cube extrusion, parameter edit, undo and redo, save and reopen, and
topology key and public face checks. See the [X11 log](native-x11-cube.log).
All eight owner placement-field tests passed, including the two inference
cases with All Geometry selected.

The scripted native cube journey passed with All Geometry selected at setup.
Its sketch re-entry comparison disables the edit-only axes in a private test
profile. It still compares the camera and viewport pixels, checks that the
rectangle visibility probe fails on hidden geometry, and verifies per-face
areas and stable keys through file reopen. See the
[frame-loop log](native-frame-loop-cube.log). Strict Clippy passed for all
desktop targets. The full workspace gate and solver precision milestone
remain open.
