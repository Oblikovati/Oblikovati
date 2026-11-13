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
#[then("the full Tools command groups are visible again")]
fn full_tools_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}

#[when("the user double-clicks the active Tools tab")]
#[when("the user double-clicks the active Tools tab again")]
fn double_click_active_tools(world: &mut ApplicationWorld) {
    std::thread::sleep(std::time::Duration::from_millis(500));
    double_click_tab(world, "Tools");
}

#[then("the ribbon shows its tabs without permanent command groups")]
fn tools_tabs_without_commands(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    world.expect_no_text("Application Options");
    world.expect_no_text("Document Settings");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
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

#[given("Tools is the active tab in tabs-only ribbon display")]
fn tools_tabs_only(world: &mut ApplicationWorld) {
    full_tools(world);
    for _ in 0..3 {
        activate_display_control(world);
    }
    world.expect_no_text("Application Options");
}

#[when("the user double-clicks the inactive View tab")]
fn double_click_inactive_view(world: &mut ApplicationWorld) {
    double_click_tab(world, "View");
}

#[then("View becomes the active tab")]
#[then("View remains the active tab in tabs-only display")]
fn view_active(world: &mut ApplicationWorld) {
    let view = world.pixel_left_of("View");
    let tools = world.pixel_left_of("Tools");
    assert_ne!(view, tools, "expected View to be selected");
}

#[then("its command groups appear in a temporary panel")]
fn temporary_view(world: &mut ApplicationWorld) {
    world.expect_text("User Interface");
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
}

#[when("the user double-clicks the active View tab")]
fn double_click_active_view(world: &mut ApplicationWorld) {
    std::thread::sleep(std::time::Duration::from_millis(500));
    double_click_tab(world, "View");
}

#[then("the full View command groups are visible")]
fn full_view_visible(world: &mut ApplicationWorld) {
    world.expect_text("User Interface");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
