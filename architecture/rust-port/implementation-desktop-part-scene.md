# Owned desktop part scene publication — partial R1V/R4

The desktop now publishes the active document's current ordered datum/prism
result through its existing file-owned session. The sealed request, result
adoption checks, and retained body storage from
[part adoption](implementation-part-adoption.md) remain the only result path.
Kernel bodies stay on the desktop owner thread. The window receives only owned
display triangles and a generation-tagged status value.

Scene publication runs after the owner and script services have polled. This
prevents a script edit from publishing an earlier scene in the same frame.
An unchanged document/generation produces no replacement and no kernel replay.
Changing focus retires the previous document's result slot. Closing the active
document publishes an empty scene. Owner retirement drops retained bodies before
the native window is closed.

Each successful current body uses the released kernel's existing copy-and-mesh
path. No application mesher or modeling approximation was added. A local display
conversion decline names its feature and remains in presentation diagnostics.
Other successful bodies still contribute geometry. Failed, blocked, suppressed,
and cutoff features contribute no current body geometry.

The native window retains ordered feature health and display declines for UI
consumers. Its actual X11 window title shows the displayed document, generation,
solid count, and unavailable-feature count. A failed input replaces old geometry
with the current successful subset; an entirely failed part becomes empty with
visible failure status. This does not present last-good geometry as current.

Tests load a saved prism package through the canonical typed API, change width,
undo, redo, fail depth, and close. Native GPU captures match actual X11 pixels.
Undo and redo reproduce the prior pixel bytes, and failed/closed states contain
no old solid pixels. A separate owner test keeps an independent body visible
beside a failed feature. Mesh-option declines retain their feature identity and
leave analytic reports unchanged. The frame loop refuses a window that cannot
accept model scenes, rather than silently dropping an update.

This remains the stored selected-prism capability. Complete extrude behavior,
general feature coverage, incremental replay, background scheduling, camera
interaction, picking, and the full UI remain open. The native title is a current
status surface, not the completed feature-health panel or public diagnostic API.
Required Go kernel extensions still belong in Martensite.

Run `CARGO_NET_OFFLINE=true make gate` for the full local Linux checks.
