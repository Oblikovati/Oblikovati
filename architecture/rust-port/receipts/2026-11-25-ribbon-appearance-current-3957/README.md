# Current native ribbon appearance

All 18 current scenarios passed individually, with 201 steps. The
published Part covered appearance modes, expanded groups, panel titles,
menu dismissal, tab switching, and Reset Ribbon confirmation. The
published empty XY sketch Part covered changing appearance during sketch
editing and switching tabs without changing the sketch.

The builds used base revision `0479c942e1f15b808a6cee9902e07f7aaaebf2aa`, the
published fixture bindings, and existing local GUI filesystem, opening,
and saving edits. The native window ran on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. The feature and fixture hashes matched the shared
source and isolated copy before and after the runs. Strict build and
runtime provenance remains open under #3978. See the
[combined report](cucumber.json), [scenario inventory](scenario-inventory.json),
and [input hashes](current-inputs.sha256).
