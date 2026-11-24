# Symmetric extrusion direction

At `1b9933b96`, the typed regression refused the normalized `Symmetric`
direction before creating a feature. The new direction keeps one positive
Distance A and starts the sweep half that distance behind the source plane.
The Go reference uses the same `[-A/2, +A/2]` span for a symmetric distance
extrusion.

The typed regression checks the analytic volume and centroid at 10 and
20 mm, after Undo and Redo, and after reopening. The native fixture selects
Symmetric through an X11 click and checks the submitted preview reaches
5 mm to either side of the XY sketch plane while the feature list stays empty.

All five steps of pinned `extrusion-03a` passed under Xvfb and Openbox. The
native companions passed for Default creation and history and Flipped
preview and commit. The eight Extrude dialog tests and six ribbon interaction
tests passed. The final owner suite passed all 27 extrusion tests and
Clippy passed for the changed targets.

The scenario checkout remained clean at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
See the [Cucumber report](pinned/cucumber.json) and
[symmetric preview](pinned/symmetric-preview.png).
