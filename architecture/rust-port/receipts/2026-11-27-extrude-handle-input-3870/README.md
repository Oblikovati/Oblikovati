# Keep the Extrude handle outside Properties

The X11 reproduction showed the handle drawn over the Properties footer.
A desktop pointer drag left the Distance A draft at 10 mm. The handle
now clears the live panel bounds. Moving the projected axis together
with the handle preserves the distance-to-pointer ratio.

The published regression checks that the handle and Properties bounds
are separate. It drags the handle to about 15 mm, checks the field,
dimension label, and preview depth, then enters 8 mm and verifies the
viewport follows. The X11 case drives the desktop pointer directly.

All seven steps passed on each Linux backend.
[Wayland output](wayland.xml), [X11 output](x11.xml),
[published input checksums](published-inputs.sha256).
