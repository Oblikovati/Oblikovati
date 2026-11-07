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
    world.expect_text("Parameters");
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
    world.click_text("Ribbon Display");
    world.pump(Vec::new());
    world.expect_text("Create");
    world.expect_text("Modify");
    world.click_text("Create");
    world.pump(Vec::new());
    world.expect_text("Extrude");
    world.close();
}
