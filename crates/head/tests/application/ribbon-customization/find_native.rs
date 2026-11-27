// SPDX-License-Identifier: GPL-2.0-only
//! The Tools ribbon and browser share the same Find dialog.
use super::*;

#[test]
fn tools_find_works_with_the_model_browser_hidden() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Model Browser");
    world.expect_no_text("Origin");
    world.escape();
    world.click_text("Tools");
    let command = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Find")
        .min_by(|left, right| left.center()[1].total_cmp(&right.center()[1]))
        .expect("Find: expected the ribbon command")
        .center();
    world.click(diagnostics::pointer_pixels(command));
    world.expect_text("Find: Features");
    world.expect_text("Find Now");
    world.click_text("Close");
    world.expect_no_text("Find: Features");
    world.expect_no_text("Origin");
    world.close();
}

#[test]
fn browser_menu_find_still_opens_the_shared_dialog() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::super::super::workspace_layout::dock_search::click_model_menu(&mut world);
    world.click_text("Find");
    world.expect_text("Find: Features");
    world.expect_text("Find Now");
    world.click_text("Close");
    world.expect_no_text("Find: Features");
    world.expect_text("Origin");
    world.close();
}
