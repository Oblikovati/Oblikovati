// SPDX-License-Identifier: GPL-2.0-only
//! Live steps for selection in a sketch's customization dialog.
use super::*;

#[given("a part document is in sketch edit in The MCAD Software")]
fn active_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
}
#[given("the ribbon shows the Sketch tab")]
fn sketch_ribbon(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.expect_text("Sketch");
}
#[when("the user opens the Create panel title context menu")]
fn open_create_menu(world: &mut ApplicationWorld) {
    let center = world.run("Create").center();
    world.right_click(diagnostics::pointer_pixels(center));
}
#[then("the selected target is \"Part | Sketch\"")]
fn selected_sketch_target(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Sketch"));
}
#[then("the source category is \"All Commands\"")]
fn all_commands_source(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("AllCommands"));
}
#[then("the \"2D Sketches\" row is highlighted")]
fn source_row_selected(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    assert!(world.row_accented_beside("2D Sketches", "<Separator>"));
}
#[then("the right transfer button is enabled")]
fn transfer_is_enabled(world: &mut ApplicationWorld) {
    assert_eq!(world.run_in_window("Customize", ">").alpha, u8::MAX);
}
#[then("Apply and Cancel remain disabled")]
fn no_pending_changes(world: &mut ApplicationWorld) {
    for label in ["Apply", "Cancel"] {
        assert_ne!(world.run_in_window("Customize", label).alpha, u8::MAX);
    }
}
#[then("no command is added to the custom panel")]
fn no_custom_panel(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
#[then("the sketch remains in edit mode")]
fn sketch_still_editable(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
