# Asymmetric extrusion direction

At `e371b388a`, the typed regression refused `secondDistance` before
creating a feature. Asymmetric extrusion now sweeps from `-B` to `+A`
relative to the resolved sketch plane. The desktop starts with Distance B
at 5 mm and uses the same requests for preview and acceptance.

The owner regression checks a 10 × 10 mm profile with an 8 mm Distance A
and 4 mm Distance B. The independent expected volume is 1,200 mm³ with
its centroid 2 mm along the sketch normal. Changing B to 6 mm gives
1,400 mm³ and a 1 mm centroid. The test exercises invalid partial edit
rejection, Undo, Redo, and reopening.

The native edit test changes Distance B, rejects zero, swaps A and B
through the visible control, commits, and checks the analytic centroid
before and after one Undo. Taper B is visible and disabled. The distinct
taper values in `extrusion-22` still require
[Martensite #463](https://github.com/Oblikovati/Martensite/issues/463).

All six steps of pinned `extrusion-03b` passed under Xvfb and Openbox.
The native edit test, 28 extrusion owner tests, 77 automation tests,
eight Extrude dialog tests, six ribbon interaction tests, and Clippy passed.

The pinned checkout remained clean at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
See the [Cucumber report](pinned/cucumber.json),
[asymmetric preview](pinned/asymmetric-preview.png), and
[native edit result](native-edit.log).
