# Desktop extrusion reference controls

The From and Termination controls expose From–To and To Face. The panel
clears a numeric preview when switching to missing references. Returning
to Distance restores the original numeric draft. A reference slot takes
its face key from the next model hit while the slot is armed, and OK
requires an accepted preview for those exact keys at the active generation.

The pinned source is `b576445b98cffd686dcc05aa8c017de2a591c8c8` and
the extrusion feature SHA-256 is
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
The [typed extent receipt](../2026-11-24-extrusion-reference-extents-3893/README.md)
contains the baseline for both desktop scenarios.

Pinned `extrusion-10` and `extrusion-11` passed all 14 steps under Xvfb
and Openbox. The native companion clicked the top face of a committed
solid, checked that its reference draft qualified for OK, and committed
a second body with a To Face recipe. The `extrusion-03c` direction
regression, 31 extrusion-related tests, six ribbon tests, and Clippy
also passed.

[From–To report](from-to/cucumber.json), [To Face report](to-face/cucumber.json),
and [viewport pick test](native-pick.log).
