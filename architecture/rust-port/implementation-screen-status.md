# Native command feedback overlay — partial R1V/R7

Save, Undo, and Redo feedback now appears inside the Linux window through
released Martensite text primitives. The existing renderer owns the device,
font lookup, atlas images, submission, native presentation, and completion.
The application supplies an owned label and screen placement through a narrow
viewport method. Dear ImGui controls and their clipping/input contract remain
open work under ADR-0008 and ADR-0009.

The overlay uses `TMF_2D`, a top-left pixel anchor, and the kernel's top OSD
layer. Depth test and depth writes are disabled for that layer. Its structure
is infinite and its text contributes no model bounds. Fit/Home and modeling
never consume the overlay as geometry. The current local font is DejaVu Sans;
font asset packaging, DPI policy, and the full application theme remain open.

## State and retirement

Overlay changes have a separate generation in viewport captures. A label
change invalidates the previous capture without changing model, camera, or
surface generation. An identical label is a no-op and preserves the completed
capture. An empty label clears the overlay. NUL text is refused before the
retained label or capture changes.

The replacement structure is built before the current structure is removed.
An owned structure guard removes a failed candidate, replaced label, or
retired view. The view's overlay retires before the kernel view and native
window. GPU lifetime stays in the kernel's completed submission protocol;
there is no elapsed-frame retirement delay or application Vulkan allocation.

## Local Linux evidence

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true make gate
```

CPU tests inject named recording graphics and font providers. They verify the
textured glyph draw and depth policy, separate generation, unchanged capture,
invalid-label and exhausted-generation failure, replacement, clear, and close.
Native tests use the real font and Vulkan device, compare native window pixels
with the completed capture, and prove that text stays fixed during pan. They
also verify that clearing text does not change Fit's camera bounds.

The document history test first checks the visible feedback, then explicitly
clears the overlay to retain its exact whole-model image comparison. Thus the
new text does not replace the existing recipe/geometry restoration oracle.

A local preview displayed a green triangle and the label
`Saved document 1: /tmp/part.opd` in the same native view. Its captured pixels
matched the X11 window and the image was inspected. The disposable harness is
in `experiments/native-status-preview`; its screenshot is a local artifact.
This proves screen-space kernel text, not the full application GUI acceptance.
No push, PR, remote test, or deferred-platform test is included.
