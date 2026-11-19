// SPDX-License-Identifier: GPL-2.0-only
//! Pinned Constrain expansion through sketch edit.
use super::*;

#[given("the MCAD Software has an active part containing an existing sketch")]
fn part_with_existing_sketch(world: &mut ApplicationWorld) {
    part_with_sketch(world);
}
#[when("the user opens the Constrain panel additional area")]
fn open_additional(world: &mut ApplicationWorld) {
    open_constrain(world);
}
#[then("Edit Coordinate System and Constraint Inference Scope are unavailable")]
fn commands_are_unavailable(world: &mut ApplicationWorld) {
    commands_unavailable(world);
}
#[when("the user clicks the unavailable Edit Coordinate System command")]
fn click_unavailable_command(world: &mut ApplicationWorld) {
    let command = world.run_in_window(WINDOW, "EditCoordinateSystem");
    world.click(diagnostics::pointer_pixels(command.center()));
}
#[then("the additional area remains open and no command starts")]
fn unavailable_command_is_ignored(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}
#[when("the user pins the additional area")]
fn pin_area(world: &mut ApplicationWorld) {
    super::super::toggle_pin(world);
}
#[when("the user opens the existing sketch for editing")]
fn edit_existing_sketch(world: &mut ApplicationWorld) {
    edit_sketch(world);
}
#[then("the Constrain additional area remains open and pinned")]
fn constrain_stays_pinned(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    super::super::pin_highlight(world);
}
#[then("Edit Coordinate System and Constraint Inference Scope are available")]
fn commands_are_available(world: &mut ApplicationWorld) {
    for label in ["Edit Coordinate System", "Constraint Inference Scope"] {
        assert!(world.run_is_enabled(label), "{label}: expected available");
    }
}
#[when("the user finishes the sketch without editing it")]
fn finish_without_edit(world: &mut ApplicationWorld) {
    finish_sketch(world);
}
