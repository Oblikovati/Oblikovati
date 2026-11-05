# Native GUI composition — partial R1V/R4

Tracking: [native composition #3630](https://github.com/Oblikovati/Oblikovati/issues/3630)
and [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
The application consumes kernel v0.2.7 at
`d40ed71b704efd08310a4c56a1c1c3a64d4e3e9b`.

The viewport and native head now accept the existing owned GUI frame packets.
`submit_gui_frame` uses the retained `GuiFrameCompiler`, then attaches its layer
to the existing `Graphic3dRenderView`. `clear_gui_frame` removes that layer.
The kernel owns its packet, atlas, device, swapchain, capture and retirement.
No additional renderer, offscreen-copy presentation or model structure is added.

Both operations use the existing overlay generation path. A successful change
invalidates capture without changing the model or camera generation. Malformed
geometry, invalid clips, wrong target extent and exhausted generations leave the
previous packet intact. A failed redraw cannot return the old capture as current.
The kernel retains the compiled atlas after the application compiler, input packet
and Dear ImGui context have been destroyed. GUI content does not enter camera Fit.

`drawable_extent` observes the actual target before the next GUI frame is built.
It shares the existing resize handling, including camera aspect, surface generation
and capture invalidation. A requested window size is not the actual size: the AMD
native test observed a 401-pixel target after a 400-pixel request. The corrected
consumer test builds its fresh packet from the observed extent. A changed extent
requires a new GUI packet; the old packet is refused rather than rescaled.
The viewport retains the surface revision of each accepted GUI packet. A resize
observed before draw or after a completed draw returns `SurfaceChanged` before
capture. The native GUI owner then publishes a fresh packet, including when the
target returned to its earlier extent. A resize that causes a kernel renderer
error during draw/capture still propagates; a typed lifecycle outcome is needed
before that error can be distinguished from a real renderer fault.

## Consumer evidence

Recorded-backend tests check ordered composition with model content, capture
invalidation, failed redraw, atomic refusal, generation exhaustion, closed views,
external target resize and the requirement for a fresh packet. The shared quad
fixture also preserves the existing standalone GUI compiler and GPU tests.

Native viewport tests check all four clip edges and alpha against independent
pixel values, ordered replacement, failed replacement, clearing, resize,
unmap/remap, compiler destruction, Fit and close. Every completed GPU capture
is compared with the actual X11 drawable. Tests pass on software Vulkan with
Xvfb and on AMD RADV with the native Xwayland display, with validation enabled.
The AMD driver cannot present to this Xvfb target; its acceptance uses the native
display, not a skipped native assertion.

A native head test opens the stored diagnostic part, creates the real Dear ImGui
panel, destroys the GUI context and compiler, and presents the retained GUI over
the model. Model pixels remain unchanged outside the panel. Clear restores the
original capture exactly. A second attachment remains valid until window close.
The test passed on both drivers. The software capture was visually inspected at
`/tmp/native-gui-composed.png`: text orientation, font coverage, button content,
clipping and model presentation are correct. Temporary image-write code was
removed before the final gate.

The focused AMD receipts are `/tmp/native-gui-amd-display-viewport-final.log`
and `/tmp/native-gui-amd-display-head-final.log`. These are consumer composition
checks, not a claim that the complete application suite passed on AMD.

## Remaining scope

The executable does not yet schedule these GUI frames or route native events to
Dear ImGui. Pointer and keyboard capture, native key/text decoding, queued-input
frame scheduling, focus handling, Save/Undo/Redo/Home dispatch, and the full Go
panels remain open. The existing status overlay remains in use. This composition
boundary does not complete R1V/R4 or clear the kernel priority item's full native
command acceptance requirement. It establishes the released renderer path on
which those consumers can be built.

Cross-platform and coherence review: production changes add no numeric constant,
platform path or geometry algorithm. Linux-only methods stay in the existing
native modules; the shared owned packet types are unchanged. Camera, clip-chain
and preview Fit comments remain. Windows and macOS acceptance remains deferred.

## Full local gate

`make gate` passed on the final source for all 13 application crates, all targets
and features. It covered formatting, Clippy, build, tests, doctests, native display,
coverage, Markdown, documentation links, SPDX, workflow checks and duplication.
The complete suite used Xvfb with software Vulkan. The retained log is
`/tmp/native-gui-composition-gate.log`.

Every crate exceeded 80% line coverage. Viewport reached 94.33%, head 96.20%
and GUI 98.89%. Duplication remains 0.77% (577 of 74,829 scanned lines,
67 pairs), with no added clone pair. Integration is local only; no push,
PR or publication is part of this change.
