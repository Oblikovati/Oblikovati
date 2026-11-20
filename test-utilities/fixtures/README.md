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
