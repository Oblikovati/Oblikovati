# Native sketch fixtures

`empty-planar-sketch-right.opd` is a native YAML part with one sketch named
`Sketch1` on the YZ origin plane. It has a projected origin and no curves.
Start sketch edit to enter Right view. Snap to Grid is an application setting;
set it separately when a scenario requires it to be off.

The native test `native_empty_right_sketch_fixture_survives_save_and_reopen`
checks the stored sketch and reopens it in the application. Its optional
`OBK_EXPORT_RIGHT_SKETCH` output must name a new file; the test will not
replace an existing file.

`docking-visible-solid.opd` is a native YAML part with a rectangular extruded
solid and its visible profile. It is a candidate for docking scenarios that
need a populated viewport. The native test
`docking_solid_and_visible_sketch_survive_save_and_reopen` checks the saved
file and the checked-in fixture. Its optional `OBK_EXPORT_DOCKING_FIXTURE`
output must name a new file.

`planar-sketch-empty-xy.opd` and `constraint-settings-line.opd` are candidates
for Constraint Settings. Each has one XY sketch named `Sketch1`. The second
has one 40 mm line with a Horizontal relation and no dimensions; the first
has no curves. Preferences are session state and must be set separately.
The native test `constraint_settings_native_fixtures_reopen_with_expected_geometry`
checks the saved and checked-in files.

## Document Saving

[SavedEmptyPart.opd](document-saving/SavedEmptyPart.opd) is a saved empty
metric Part. The Save As folder uses a different empty Part,
[ExistingPart.opd](document-saving/SaveAsFolder/ExistingPart.opd), and
[ExistingSolidPart.opd](document-saving/SaveAsFolder/ExistingSolidPart.opd),
a rectangular 40 × 25 × 20 mm solid with no assembly reference.

Copy the two Save As files to a writable scratch folder and create an empty
child directory named `Subfolder`. Use those copies for file selection and
replacement confirmation. Verify that declining replacement leaves both
files intact.

The native regression
`document_saving_native_fixtures_reopen_and_survive_declined_replacement`
opens all three files through the desktop, verifies the solid's physical
properties, then declines replacing the empty Part.

## Viewport Navigation Wheel

[main-window-frame-start.opd](main-window-frame-start.opd) is the published
scenario fixture with LF line endings. It contains a 40 × 25 × 20 mm solid and its
constrained profile. The older pinned wheel scenarios describe the block
without specifying a file. Their tests use this copy. When the published
scenario directory is supplied, its fixture is used.
