# Pick multiple Extrude profiles in the viewport

A New Solid draft can select additional closed regions of its source sketch.
Profiles shows the number of distinct regions and the temporary viewport scene
contains each solid. Clear removes the full selection and disables OK until the
next valid pick. Repeated clicks on an already selected region keep one entry.
The native owner scopes picks to the current document, sketch, and draft request.

OK publishes the solids together in one Extrude. The independently named
results use the [grouped Part recipe](../2026-11-27-extrude-grouped-replay-3870/README.md).
The published rectangle-and-circle and two-rectangles scenarios drive the
viewport picks. A native regression also accepts a two-solid draft and reopens
the saved document.

[Native and regression results](tests.txt).
