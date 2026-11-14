# Native sketch tool commands

Issue #3801 connects the ribbon to the host's existing Point, Circle,
center-point Arc, Ellipse, Rectangle, Center Rectangle and Three-point Rectangle
placement engines. The GUI sends one typed tool choice. The desktop owner selects
the corresponding engine and calls the shared placement entry point. The old
rectangle-only GUI action and host method are removed.

The large Rectangle control remains beside a compact two-column group of the
other six choices. Finish Sketch is in the Exit panel. Pointer tests cover every
choice and Finish at normal and double framebuffer scale, including a 640-pixel
logical window width. Script and add-in wrappers forward the same tool choice.

Selecting a tool starts transient placement without adding a history entry.
Switching tools clears the prior preview. Escape cancels transient placement.
Existing typed dimension fields and parameter formulas remain in the owner
pipeline. Tools that require caller options, such as polygon side count and slot
width, need their settings controls before they can join this ribbon group.

The native mouse journey creates and cancels a circle preview, commits a diameter
driven by `size`, checks circle identity through undo/redo, and creates then undoes
a point. It restores the original healthy DOF=0 rectangle before changing size,
saving and reopening. The temporary test geometry is absent from the saved Part.
The full local gate and the exact integrated native build remain the publication
checks.

Full Go command and icon parity, geometric constraint tools, content-area layout,
Extrude and the complete cube journey remain open.
