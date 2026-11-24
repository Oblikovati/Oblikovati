# Distance A editor menu

The desktop Extrude draft now exposes Measure, Select Feature Dimension,
Parameters, Tolerance for an editable length parameter, and List Parameters.
The table's Consumed By column comes from current feature and parameter reads.
Opening these tools leaves the draft in the Extrude panel. Choosing an
expression affects only the draft until confirmation.

The pinned source revision is `b576445b98cffd686dcc05aa8c017de2a591c8c8`
and the extrusion feature SHA-256 is
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
At `9f04f00db`, all four pinned Distance A editor cases stopped at their
first scenario-specific step because the matching bindings were absent.
See the [baseline report](baseline.json).

Pinned `extrusion-12a`, `extrusion-12b`, and `extrusion-12c` passed all
26 steps under Xvfb and Openbox. The native companion selected a second
feature and inserted its current length parameter, then canceled and
verified the first extrusion still measured 8 mm. The `extrusion-03c`
regression, 89 parameter tests, six ribbon interaction tests, and Clippy
passed.

[Editor menu report](menu/cucumber.json),
[feature dimension report](feature-dimension/cucumber.json),
[parameter table report](parameters/cucumber.json),
[native dimension pick](native-dimension.log).
