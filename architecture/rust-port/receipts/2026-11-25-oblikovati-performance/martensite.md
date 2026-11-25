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
waiting for pixels. For a changed live tab preview and save thumbnail, it
uses the existing packed RGB dump and reuses the destination pixel map. At
1440×900 this holds 3,888,000 bytes, compared with 10,368,000 bytes for the
full RGBA and depth planes. Each 64×36 thumbnail needs 6,912 bytes of RGB
samples from the workspace rectangle. Explicit screenshots need full RGBA.
[Packed RGB measurements](../2026-11-25-interactive-preview-rgb/README.md)

The application now reuses a tab image on repeated presents of identical
composition generations and crop regions. The smaller transfer is still
needed when the camera, overlay, or source scene changes. The application
also retains unchanged compiled GUI geometry across changing layers.
[Application evidence](../2026-11-25-render-reuse/README.md)

Expose a typed outcome for the packed RGB route so callers can distinguish
resize, generation mismatches, and renderer faults without attempting full
capture. Add a region-to-thumbnail capture that samples the accepted native
publication under the same ownership checks. Validate generation and resize
before and after the transfer, preserve device-loss errors, and match the
current top-left sRGB ordering. Compare pixels against full capture with
non-square extents, several crops, resize, and two concurrently open views.

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
With packed RGB in use, face hover and selection still show p95 intervals
of 16.52–23.51 ms. Include this workload in the phase comparison.

## Attribute X11 window observations

The [post-reuse orbit sample](../2026-11-25-render-reuse/README.md) reports
6.70% inclusive user-space cycles in `XGetWindowAttributes`. Sample stacks
reach it through `XwWindow::live_attributes`. The sample neither counts
round trips nor captures wall time blocked waiting for replies.

The subsequent [window-observation audit](../2026-11-25-x11-window-observations/README.md)
verified an application-side reduction. The updated build still made
approximately 64 attribute calls per unchanged present and 206–208 per
orbit present in the measured runs. These are whole-process counts and
include work in the application, renderer, and Vulkan driver.

Count and time these calls per redraw, separating application extent
queries from checked GUI attachment, resize checks, and swapchain
publication. Investigate sharing one versioned observation where its
lifetime permits it. Verify externally delivered resizes, unmap/remap, and
surface replacement between acquisition and presentation before changing
when an observation may be reused.
