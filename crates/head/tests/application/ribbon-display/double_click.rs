// SPDX-License-Identifier: GPL-2.0-only
//! Real pointer input for ribbon tab double-click scenarios.
use super::*;

fn double_click_tab(world: &mut ApplicationWorld, label: &str) {
    let position = super::super::diagnostics::pointer_pixels(world.run(label).center());
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        position, 2,
    ));
    world.pump(Vec::new());
}

#[when("the user double-clicks the inactive Tools tab")]
fn double_click_inactive_tools(world: &mut ApplicationWorld) {
    double_click_tab(world, "Tools");
}

#[then("the full Tools ribbon is visible")]
#[then("the full Tools ribbon is visible again")]
fn full_tools_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}

#[when("the user double-clicks the active Tools tab")]
fn double_click_active_tools(world: &mut ApplicationWorld) {
    std::thread::sleep(std::time::Duration::from_millis(500));
    double_click_tab(world, "Tools");
}

#[then("the Tools ribbon displays panel buttons")]
fn tools_panel_buttons(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    world.expect_no_text("Application Options");
}

#[when("the user double-clicks the active Tools tab three more times")]
fn cycle_tools(world: &mut ApplicationWorld) {
    for _ in 0..3 {
        double_click_active_tools(world);
    }
}
