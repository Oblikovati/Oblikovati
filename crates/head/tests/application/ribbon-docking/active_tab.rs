// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{given, then, when};

#[given("the full Manage ribbon is docked at the top")]
fn manage_ribbon_at_top(world: &mut ApplicationWorld) {
    world.click_text("Manage");
    world.pump(Vec::new());
    world.expect_text("Parameters");
    assert!(window_geometry(COLUMN).is_none());
    assert!(window_geometry(FLOATING).is_none());
}
#[given("panel titles are shown")]
fn titles_shown(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[given("the full 3D Model ribbon is docked at the top")]
fn model_ribbon_at_top(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.expect_text("Start 2D Sketch");
    world.expect_text("Modify");
    assert!(window_geometry(COLUMN).is_none());
    assert!(window_geometry(FLOATING).is_none());
}
#[when("the user right-clicks the active 3D Model tab")]
fn right_click_model(world: &mut ApplicationWorld) {
    let center = world.run("3D Model").center();
    world.right_click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
#[when("I right-click the active 3D Model tab and open Docking Positions")]
fn open_model_docking_positions(world: &mut ApplicationWorld) {
    right_click_model(world);
    open_docking_positions(world);
}
#[when("I right-click the active 3D Model tab and choose Left from Docking Positions")]
fn dock_left_from_model_tab(world: &mut ApplicationWorld) {
    open_model_docking_positions(world);
    world.click_text("Left");
    world.pump(Vec::new());
}
#[then("the 3D Model ribbon remains visible")]
fn model_still_visible(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
    world.expect_text("Start 2D Sketch");
    world.expect_text("Modify");
}
#[then("the Docking Positions submenu closes and the context menu remains open")]
#[then("the Docking Positions submenu closes and the title context menu remains open")]
fn docking_submenu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Left");
    world.expect_no_text("Right");
    world.expect_text("Undock Ribbon");
}
#[then("the context menu closes and the 3D Model ribbon remains docked at the top")]
fn context_closed_at_top(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
    world.expect_text("Start 2D Sketch");
    assert!(window_geometry(COLUMN).is_none());
    assert!(window_geometry(FLOATING).is_none());
}
#[when("the user right-clicks the active Manage tab")]
fn right_click_manage(world: &mut ApplicationWorld) {
    let center = world.run("Manage").center();
    world.right_click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
#[then("Show Panel Titles is checked")]
fn titles_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Show Panel Titles"));
}
#[then("the menu offers Customize User Commands, Undock Ribbon and Docking Positions")]
fn menu_commands(world: &mut ApplicationWorld) {
    for label in [
        "Customize User Commands",
        "Undock Ribbon",
        "Docking Positions",
    ] {
        world.expect_text(label);
    }
}
#[then("the Manage ribbon remains visible")]
fn manage_still_visible(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
    world.expect_text("Parameters");
}

fn parameter_runs(world: &ApplicationWorld) -> usize {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Parameters")
        .count()
}
#[test]
fn panel_titles_toggle_from_active_tab_context_menu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    manage_ribbon_at_top(&mut world);
    let shown = parameter_runs(&world);
    right_click_manage(&mut world);
    titles_checked(&mut world);
    world.click_text("Show Panel Titles");
    world.pump(Vec::new());
    assert_eq!(parameter_runs(&world) + 1, shown);
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.expect_text("Start 2D Sketch");
    world.expect_text("Extrude");
    world.expect_no_text("Modify");
    manage_ribbon_at_top(&mut world);
    assert_eq!(parameter_runs(&world) + 1, shown);
    right_click_manage(&mut world);
    assert!(!world.checked("Show Panel Titles"));
    world.click_text("Show Panel Titles");
    world.pump(Vec::new());
    assert_eq!(parameter_runs(&world), shown);
    right_click_manage(&mut world);
    world.click_text("Show Panel Titles");
    world.pump(Vec::new());
    world.click_text("3D Model");
    world.pump(Vec::new());
    super::super::ribbon_display::click_display_menu(&mut world);
    world.click_text("Minimize to Panel Buttons");
    world.pump(Vec::new());
    world.expect_text("Create");
    world.expect_text("Modify");
    world.click_text("Create");
    world.pump(Vec::new());
    world.expect_text("Extrude");
    world.close();
}

#[test]
fn hidden_captions_keep_narrow_collapsed_panels_named() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.app().window.native.view.resize([700, 600]).unwrap();
    world.pump(Vec::new());
    world.click_text("Manage");
    world.pump(Vec::new());
    right_click_manage(&mut world);
    world.click_text("Show Panel Titles");
    world.pump(Vec::new());
    world.click_text("3D Model");
    world.pump(Vec::new());
    let create = world.run("Create").center();
    world.click(super::super::diagnostics::pointer_pixels([
        create[0],
        create[1] - 35.,
    ]));
    world.pump(Vec::new());
    world.expect_text("Extrude");
    world.escape();
    world.pump(Vec::new());
    let modify = world.run("Modify").center();
    world.click(super::super::diagnostics::pointer_pixels([
        modify[0],
        modify[1] - 35.,
    ]));
    world.pump(Vec::new());
    world.expect_text("Hole");
    world.close();
}

#[test]
fn docking_submenu_reopens_after_escape_and_moves_the_ribbon() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    right_click_model(&mut world);
    let row = world.run("Docking Positions").center();
    world.hover(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.expect_text("Left");
    world.escape();
    world.pump(Vec::new());
    docking_submenu_closed(&mut world);
    world.click_text("Docking Positions");
    world.pump(Vec::new());
    world.expect_text("Left");
    world.click_text("Left");
    world.pump(Vec::new());
    assert!(window_geometry(COLUMN).is_some());
    world.expect_no_text("Undock Ribbon");
    world.close();
}

#[test]
fn docking_submenu_opens_and_closes_with_arrow_keys() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    right_click_model(&mut world);
    for _ in 0..8 {
        world.key(ViewportPhysicalKey::DownArrow, false);
        world.key(ViewportPhysicalKey::RightArrow, false);
        if world.shows("Left") {
            break;
        }
    }
    world.expect_text("Left");
    world.key(ViewportPhysicalKey::LeftArrow, false);
    docking_submenu_closed(&mut world);
    world.close();
}
