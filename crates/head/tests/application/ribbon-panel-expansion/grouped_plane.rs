// SPDX-License-Identifier: GPL-2.0-only
//! Moving the grouped Plane control between ribbon areas.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

const EXPANSION: &str = "##ribbon-additional-PartWorkFeatures";

#[given("the grouped \"Plane\" control is in the 3D Model Work Features main panel")]
#[given("the 3D Model Work Features panel contains a grouped \"Plane\" command")]
fn plane_in_main_panel(world: &mut ApplicationWorld) {
    world.expect_text("Work Features");
    world.expect_text("Plane");
}

#[when("the user clicks the Plane drop-down arrow")]
fn open_plane_variants(world: &mut ApplicationWorld) {
    let plane = world.run_in_window("##application-ribbon", "Plane");
    let arrow = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == plane.list && run.text == "v")
        .filter(|run| {
            run.minimum[0] > plane.maximum[0] && (run.center()[1] - plane.center()[1]).abs() < 18.
        })
        .min_by(|left, right| left.minimum[0].total_cmp(&right.minimum[0]))
        .expect("expected the arrow beside Plane");
    world.click(diagnostics::pointer_pixels(arrow.center()));
    assert!(diagnostics::command_popup_open());
    world.expect_text("Three Points");
}

#[when("the user opens the Plane drop-down and selects \"Offset from Plane\"")]
fn select_offset_from_plane(world: &mut ApplicationWorld) {
    open_plane_variants(world);
    world.click_text("Offset from Plane");
    assert!(!diagnostics::command_popup_open());
}

#[then("the software prompts the user to select a planar face")]
fn offset_waits_for_face(world: &mut ApplicationWorld) {
    world.run_in_window("Offset Plane", "SelectPlane");
    assert!(!world.run_is_enabled("OK"));
}

#[when("the user presses Escape without selecting a face")]
fn cancel_offset_before_pick(world: &mut ApplicationWorld) {
    world.escape();
}

#[then("the software returns to the ready state")]
fn offset_command_closes(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Offset Plane").is_none());
    assert!(!world.state().transaction_open);
}

#[then("the grouped Plane control and the part remain visible")]
fn plane_and_part_remain_visible(world: &mut ApplicationWorld) {
    plane_in_main_panel(world);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("no new work plane is created")]
fn no_work_plane_created(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected the Part browser");
    assert!(
        browser
            .features
            .iter()
            .all(|feature| feature.kind == "Extrusion")
    );
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
