# Extrusion distance tolerance

Tolerance uses the authored length and the document's units to display
the evaluated size. The editor offers the twelve formats in the pinned
scenario, along with Precision, evaluated-size selection, working-unit
deviations, and ISO Hole and Shaft classes. Cancel keeps the extrusion
draft and stored parameter as they were.

The native source is `b576445b98cffd686dcc05aa8c017de2a591c8c8` with
extrusion feature SHA-256
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
The earlier [Distance A editor baseline](../2026-11-24-extrusion-distance-editor-3893/baseline.json)
records the missing tolerance scenario bindings.

Pinned `extrusion-12` passed all ten steps under Xvfb and Openbox. The
owner regression verified the resolved H7 fit, a precision change in the
same history step, and Undo and Redo. Seven owner tolerance tests, 90 GUI
parameter tests, the model library tests, and Clippy passed.

[Cucumber report](pinned/cucumber.json), [tolerance dialog](pinned/distance-tolerance.png),
[format list](pinned/distance-tolerance-types.png), and [owner result](owner-tolerance.log).
