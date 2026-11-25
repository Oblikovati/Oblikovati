# Martensite v0.14.1 renderer performance follow-up

Oblikovati's X11 viewport consumes checked native GUI publication from
Martensite v0.14.1 at `0e72f866844196ae656b23a401c8b81f77b10491`. This report
covers the renderer boundary. The results and instrumentation are in the
[Oblikovati performance receipt](README.md).

## Expose typed color and thumbnail capture

`Graphic3dRenderView::capture` requests both planes and clones the frame.
`with_capture` accepts a plane selection and is crate-private. The public
`Graphic3dCView::buffer_dump` returns a boolean and logs the render error
rather than returning the checked GUI publication outcome.

Oblikovati's normal X11 present now avoids readback when no UI consumer is
waiting for pixels. A visible document still has a live tab preview and save
thumbnail. Those paths need 64×36 color samples of the workspace rectangle
from the accepted frame. Explicit screenshots need full RGBA. At 1440×900
the combined `Vec<u8>` RGBA and `Vec<f32>` depth contain 10,368,000 bytes per
owned full-plane capture.

The application now reuses a tab image on repeated presents of identical
composition generations and crop regions. The smaller transfer is still
needed when the camera, overlay, or source scene changes. The application
also retains unchanged compiled GUI geometry across changing layers.
[Application evidence](../2026-11-25-render-reuse/README.md)

Provide a typed color-only capture or borrow against the most recent
accepted native publication. Add a region-to-thumbnail capture for color
previews if the color-only route is still significant. Validate generation
and resize before and after the transfer, preserve device-loss errors, and
match the current top-left sRGB pixel ordering. Prove the pixel result
against the existing full capture with non-square extents, several crops,
resize, and two concurrently open views.

## Measure native frame preparation and completion

The present probe times acquisition, submission, fence waits, and
presentation with the same monotonic clock. The preceding X11 investigation
found long gaps mostly outside those calls. These are application-level
observations and leave the renderer's CPU preparation and host scheduling
unattributed. [X11 phase traces](../2026-11-25-x11-post-dpi-performance/README.md)

Instrument the same accepted native frame at these boundaries:

- Measure checked GUI layer consumption, retained scene extraction,
  `OwnedWindowFrame::new`, command recording, and bytes staged for upload.
- Distinguish queue handoff, readback-ticket settlement, GPU completion, and
  host pixel ownership. Record whether each plane was copied or read.
- Compare stationary camera, camera-only redraw, selection-overlay update,
  and GUI texture or geometry change. Record cache hits, allocation bytes,
  CPU elapsed time, and GPU timestamp intervals.

Use consecutive presents to derive p50 and p95 frame intervals at the same
1440×900 physical size on the same device. Compare FIFO with explicitly
requested immediate mode, and record the selected mode with validation
on and off. The 120 FPS interval budget is 8.33 ms. Attribute a stage before
changing cache or synchronization policy so correctness and resource lifetime
can be verified against the same pixel and resize oracles.

## Attribute X11 window observations

The [post-reuse orbit sample](../2026-11-25-render-reuse/README.md) reports
6.70% inclusive user-space cycles in `XGetWindowAttributes`. Sample stacks
reach it through `XwWindow::live_attributes`. The sample neither counts
round trips nor captures wall time blocked waiting for replies.

Count and time these calls per redraw, separating application extent
queries from checked GUI attachment, resize checks, and swapchain
publication. Investigate sharing one versioned observation where its
lifetime permits it. Verify externally delivered resizes, unmap/remap, and
surface replacement between acquisition and presentation before changing
when an observation may be reused.
