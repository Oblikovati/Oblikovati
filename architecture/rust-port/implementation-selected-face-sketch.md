# Selected-face sketch command

The Part marking menu exposes `New Sketch` when one current named face is
selected. It carries the document and face key to the desktop owner, which
checks the current selection and resolves that exact name before creating an
associative face-based sketch. Missing, stale, mixed, and changed selections
are refused. With no selected face, the existing sketch command continues to
request a plane. The support must resolve to a valid planar sketch plane.

The native pointer diagnostic selects a face, opens the Part marking menu,
starts the sketch, checks its face key, and saves and reopens the document.
The owner diagnostic checks a stale key, parameter replay, undo, and redo. These
checks do not establish full acceptance of the live face-context scenario:
its standard menu and other radial commands remain separate work, and the
scenario source is not pinned (#4050, #4048, #3978).
