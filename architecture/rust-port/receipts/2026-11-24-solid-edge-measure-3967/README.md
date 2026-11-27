# Solid-edge length in Part Measure

Issue #3967. The Part Measure panel accepts the existing named-edge picks and
measures the current body through the same analytic Martensite operation as
the typed length query. The adapter integrates the analytic curve with
triangulation disabled. The panel converts centimeters to millimeters and
discards rows from an earlier generation.

All six steps of pinned `extrusion-06` passed under Xvfb and Openbox. The
scenario constructs an 8 mm extrusion, accepts it,
undoes and redoes creation, and verifies the restored shaded body. The final
step opens Measure from the Tools ribbon, clicks one extrusion-depth edge,
Ctrl-clicks a second edge, and checks both rendered `Length 8.000 mm` rows.

The Tools ribbon command and viewport picks use X11 input. The sketch setup
and extrusion history use the native frame loop. The rectangular fixture
places both picked edges perpendicular to its sketch plane, so their expected
length follows directly from the authored 8 mm depth.

The focused measurement tests passed nine cases, including typed name
resolution after edit, undo, redo, and reopen, rejection after deletion, and
clearing stale panel rows. Three native companion tests passed for face area,
selection priority, and switching between Part and sketch Measure. Clippy
passed for all head targets with warnings treated as errors.

- Scenario source: `b576445b98cffd686dcc05aa8c017de2a591c8c8` (clean checkout).
- Feature SHA-256: `ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
- Earlier run with an undefined final step: [Cucumber report](../2026-11-24-history-checkpoint-4079/pinned/cucumber.json).
- Passing run: [Cucumber report](pinned/cucumber.json) and [Measure panel capture](pinned/measure-panel.png).
