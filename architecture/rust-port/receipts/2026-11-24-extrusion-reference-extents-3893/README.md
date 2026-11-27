# Reference extents for extrusions

To Face uses the profile plane and a named target. From–To uses two named
targets. The resolver measures both offsets along the profile normal and
sweeps between the lower and higher offsets. It declines nonparallel or
coincident planes before publication. Both extents use the existing certified
prism evaluator.

The owner tests use a nominal 10 × 10 mm square. A named face at 12 mm
gives 1,200 mm³ and a 6 mm centroid. Moving that face to 16 mm changes the
result to 1,600 mm³ and an 8 mm centroid. The test exercises the upstream
edit, Undo, Redo, and reopening. From–To uses planes 4 and 12 mm from the
sketch and expects 800 mm³ with an 8 mm centroid. Missing, nonparallel,
and coincident references leave the document generation unchanged. Both
extent variants round-trip through the native document package.

The 31 extrusion-related tests, 360 automation, model, persistence, and
session library tests, and Clippy passed. The pinned numeric extrusion
selector regression `extrusion-03c` passed all 12 steps under Xvfb and
Openbox. [Owner test result](owner-regression.log)

At `8cc325337`, both desktop scenarios stopped at their first
scenario-specific step because the bindings were absent. Those cases remain
open. [Extrusion-10 baseline](extrusion-10-before.json),
[extrusion-11 baseline](extrusion-11-before.json),
[pinned source revision](source-revision.json).
