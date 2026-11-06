# Checked native GUI resize

Issue [#3851](https://github.com/Oblikovati/Oblikovati/issues/3851) records a
native resize fault found during ribbon acceptance. The empty desktop exited
when its target changed after the size check and before redraw. The error named
an old GUI packet extent and the new native extent. A clean checkout of the
ribbon base, `6bf4b0dc1422978445cdc193845c1623cc544929`, reproduced the same error.

The viewport selects the released SDK's `RenderGuiSurfacePolicy::CheckedV2`.
Attachment, redraw, and capture consume typed surface-change results before
formatting errors. The host invalidates the completed capture and requests a
packet for the current target. An unchanged-size target revision also needs a
new surface generation. An already stale packet keeps that generation instead
of counting the same change twice.

The SDK's packet state determines whether a fresh packet is required, a newer
packet is already current, or GUI content is absent. The strict submission API
still refuses an old packet. The native GUI publisher uses a checked submission
result to schedule its next frame. Primary renderer faults reach the existing
error path even when the target also changes.

## Verification

The 32 recorded viewport tests passed before and after extracting the named
backend fixture. Three added tests then failed on surface changes during
redraw, capture, and a target revision with the same extent. After the repair,
all 148 viewport unit tests passed, including eight checked GUI cases. They also
cover attachment refusal, invalid draw input, primary fault retention, keeping a
newer green GUI packet, and retaining the model after GUI clear.

A native X11 test alternated widths 640, 1440, 320, and 1024 for 200 resizes at
20 ms intervals. It returned a completed MCP screenshot and closed normally.
The original ribbon journey then passed, including preview/cancel, parameter
edit, undo/redo, save, and reopening in the same and a fresh process. Captures
were inspected. The full application gate passed across all 13 crates,
including all 385 head tests. Viewport coverage was 93.40%, head coverage was
92.76%, and workspace duplication was 1.02%. The ribbon verification record
lists the cross-target compile checks for this combined change.
