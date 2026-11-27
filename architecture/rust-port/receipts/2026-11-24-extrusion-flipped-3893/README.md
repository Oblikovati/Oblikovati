# Flipped extrusion direction

The typed regression at `c4cd931a7` rejected the normalized `Negative`
direction before creating a feature. The existing prism evaluator supports
signed sweep vectors. The desktop and typed routes now retain a positive
authored depth and reverse the resolved sketch-normal component.

All five steps of pinned `extrusion-03` passed under Xvfb and Openbox. The
fixture uses a rectangular profile on XY. The
scenario compares the visible preview before and after clicking Flipped.
The separate desktop commit test compares the analytic body centroid with
`-5 mm` for a 10 mm extrusion. The typed regression checks centroid,
positive Distance A, driver edits, undo, redo, a direct distance edit, and
save and reopen.

All 26 extrusion owner tests, eight Extrude dialog tests, and six ribbon
interaction tests passed. The two native companion tests passed, covering
Flipped commit and distance validation with cancellation. Clippy passed for
the changed crates.
All six steps of `extrusion-06` passed on the same build, covering default
direction, history, and the two Measure results. See its
[Cucumber report](history/cucumber.json).

The full feature diagnostic at `c4cd931a7` found five passing scenarios,
25 with unbound steps, and one capture-only exclusion. The focused run
above closes the `extrusion-03` gap from that diagnostic.
[Baseline diagnostic](baseline/gap-report.json).

The pinned scenario tree was clean at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature hash was
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
See the [Cucumber report](pinned/cucumber.json) and
[flipped preview](pinned/flipped-preview.png).
