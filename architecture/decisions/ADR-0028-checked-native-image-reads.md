# ADR-0028: Checked native image reads

Date: 2026-10-14. Status: Accepted for the Rust target design.
Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
Boundary: the existing internal native drawable capture operation.

## Problem and deletion gate

At application base `af052abc`, an owned native window extends outside a private
display. `NativeWindow::pixels` calls XGetImage, receives BadMatch, and exits
through Xlib's process-global error path. A mapped window alone is insufficient
to guarantee a readable complete rectangle. Checking bounds before that request
would still leave a movement or resize race.

Delete the XGetImage/XImage/XGetPixel capture path. Use one checked x11rb image
reader for each native window, opened lazily on that window's actual display.
It owns no renderer, GPU resource, or model. Keep its connection and protocol
types private. No host wire or API contract changes are required.

## Decision

Read mapping state and current extent through checked requests, then request
the complete drawable image. A clipped, unmapped or retired resource returns
a named error with its identity and expected shape. Never crop the result or
change the window to make a read succeed. Do not install or replace a global
Xlib error callback. A failed image request preserves the checked connection.

Decode the server's image representation: visual and depth, RGB masks, pixel
width, image byte order, and scanline padding. Validate the complete padded
reply before indexing rows. Use checked byte-count arithmetic for 32-bit and
64-bit hosts. Preserve the existing integer channel scaling and opaque alpha.
Unsupported metadata produces a named decline. There is one decoder and no
legacy fallback.

The [X11 protocol](https://xorg.freedesktop.org/releases/X11R7.7/doc/xproto/x11protocol.html)
defines image storage independently of the request connection's byte order.
This operation reads a derived view of the B-rep. No modelling or topological
decision uses its pixels. Completed kernel framebuffers remain the GUI rendering
oracle; a successful X11 read does not establish framebuffer freshness.

## Evidence and required acceptance

The clipped-window regression exits with BadMatch at the fixed application
base. With checked reads, it returns a named decline, then produces exact known
RGB values after the owner explicitly resizes and paints the same window.
Private-display tests also cover unmap/remap, retired resources, resource-ID
overflow and reuse of a reader after a failed request.

Named fake server metadata checks 8/16/24/32-bit pixels, both image byte orders,
row and reply padding, unused high bits, truncated and oversized replies,
mismatched extents, byte-count overflow and invalid metadata. The old engine is
removed. All 132 viewport unit tests pass. The full local gate passed across
all 13 crates with minimum line coverage 91.82%, viewport coverage 93.00%,
and duplication 0.75%. The native GUI, clipboard-health, Save, resize, wheel
and keyboard checks pass on AMD. The full application milestones remain open.
