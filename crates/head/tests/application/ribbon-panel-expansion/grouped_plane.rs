// SPDX-License-Identifier: GPL-2.0-only
//! Moving the grouped Plane control between ribbon areas.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

const EXPANSION: &str = "##ribbon-additional-PartWorkFeatures";

#[given("the grouped \"Plane\" control is in the 3D Model Work Features main panel")]
fn plane_in_main_panel(world: &mut ApplicationWorld) {
    world.expect_text("Work Features");
    world.expect_text("Plane");
}
#[when("the user right-clicks \"Plane\" and selects \"Move to Expanded Panel\"")]
fn move_plane_to_expanded(world: &mut ApplicationWorld) {
    let plane = diagnostics::pointer_pixels(world.run("Plane").center());
    world.right_click(plane);
    world.expect_text("Move to Expanded Panel");
    world.click_text("Move to Expanded Panel");
}
#[then("\"Plane\" is no longer visible in the main panel")]
fn plane_leaves_main_panel(world: &mut ApplicationWorld) {
    assert!(!world.window_text("##application-ribbon").contains("Plane"));
}
#[when("the user opens the Work Features expansion")]
fn open_work_features(world: &mut ApplicationWorld) {
    super::click_expansion_arrow(world, "Work Features");
}
#[then("a \"Plane\" control appears in the expanded area")]
fn plane_in_expansion(world: &mut ApplicationWorld) {
    world.run_in_window(EXPANSION, "Plane");
}
#[then("its context menu offers \"Ungroup Commands from Drop-Down Menu\"")]
fn grouped_context_menu(world: &mut ApplicationWorld) {
    let plane = world.run_in_window(EXPANSION, "Plane");
    world.right_click(diagnostics::pointer_pixels(plane.center()));
    world.expect_text("Ungroup Commands from Drop-Down Menu");
    world.expect_text("Move to Main Panel");
}
#[when("the user selects \"Move to Main Panel\" from that context menu")]
fn restore_plane_to_main(world: &mut ApplicationWorld) {
    world.click_text("Move to Main Panel");
}
#[then("\"Plane\" returns to the main panel")]
fn plane_returns(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
    world.run_in_window("##application-ribbon", "Plane");
}
