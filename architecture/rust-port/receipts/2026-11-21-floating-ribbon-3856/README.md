# Floating ribbon scrolling and Annotate

The full published Ribbon Docking feature passed 20 of 22 scenarios on
application revision `dc6d945fdd80351e3f273fb9b80164386c3c3d25`, API v0.165.0, and Martensite v0.13.1.
The [Cucumber](cucumber.json) and [JUnit](junit.xml) reports record
193 passed steps and 2 failed steps.
The [local snapshot](snapshot.json) records the feature and fixture hashes.
The published feature SHA-256 was
`3314b980c97be4aa018b9628898ad8758179b0737ed40d4f71f75a31395bf026`. The test source remained unchanged during the run.

The X11 wheel moved Start 2D Sketch out of view and revealed the
lower modeling groups in their published order. A separate X11 click
on Start 2D Sketch entered the plane picker directly and let the user
start editing a sketch in the same Part.

Both Annotate scenarios displayed all five groups and the Notes pane.
The pane listed Leader Text, General Note, and General Profile Note as
disabled commands. Switching to 3D Model closed the pane and restored
the modeling groups while the same Part remained active.
Shape Generator and Create Freeform are also displayed as unavailable.

Both [floating X11 tests](x11-floating.log) and the
[46 GUI ribbon interaction tests](ribbon-click.log.gz) passed under Xvfb.
The [native log](native.log.gz) records the starting failure in each
remaining case. The pinned application binding check reports
1,340 unresolved step entries, down from 1,355.
The full gate remains open under [#3860](https://github.com/Oblikovati/Oblikovati/issues/3860).

The Shell scenario still lacks its fixture and preview bindings. A
separate [X11 diagnostic](shell-fixture.log) opened the published
`docking-visible-solid-current.opd`, started Shell, and selected one face
at the default 1 mm thickness. The host refused the preview because the
kernel returned a vertex tolerance of `5e-4` against a limit of `1e-7`.
The [capture](published-shell-preview.png) shows the selected face and
the error with no cutaway. This reproduces the tolerance widening tracked
in [Martensite #554](https://github.com/Oblikovati/Martensite/issues/554)
on the published Part. The mixed-document case still needs a native
Sheet Metal fixture. See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
