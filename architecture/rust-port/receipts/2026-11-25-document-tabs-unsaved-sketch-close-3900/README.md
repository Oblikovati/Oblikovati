# Closing document tabs with unsaved sketches

Four current scenarios passed all 41 steps. Cancel at the exit prompt
kept the XY sketch editor active. Cancel at the subsequent save prompt
kept the Part and its sketch open in the modeling workspace. Cancel at
Save As also kept the unsaved Part open. Accepting the suggested name
in a separate empty folder saved the Part, closed its tab, restored
PartOne, and let the saved sketch reopen.

The runs used base revision `836257d831b492a1964f514d6eb2ac673f41431a`, the unsaved-sketch bindings,
and existing local GUI filesystem, opening and saving edits. The native
window ran on Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The
[inputs](current-inputs.sha256) matched their isolated copies. Strict build
and runtime provenance remains open under #3978.
