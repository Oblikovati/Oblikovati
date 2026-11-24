// SPDX-License-Identifier: GPL-2.0-only
//! Native interactions for the shared tabs-only ribbon scenarios.
use super::*;

#[given("the Tools tab is selected in tabs-only ribbon display")]
fn tools_in_tabs_only(world: &mut ApplicationWorld) {
    full_tools(world);
    choose_tabs_mode(world);
    world.expect_no_text("Document Settings");
}

#[when("the user clicks the Tools tab")]
fn click_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
}

#[when("the user clicks the Tools tab again")]
fn click_tools_again(world: &mut ApplicationWorld) {
    // Separate clicks must not become the distinct double-click gesture.
    std::thread::sleep(std::time::Duration::from_millis(500));
    world.click_text("Tools");
}

#[then("the Tools command groups appear temporarily below the tabs")]
#[then("the temporary Tools command groups remain visible")]
fn temporary_tools(world: &mut ApplicationWorld) {
    world.expect_text("Document Settings");
    let (position, _) = window_geometry("##ribbon-temporary-panel")
        .expect("Tools commands: expected the temporary panel");
    assert!(position[1] > world.run("Tools").maximum[1]);
}

#[when("the user clicks empty viewport space")]
#[when("the user clicks empty viewport space outside the panel")]
#[when("the user clicks an empty area of the viewport")]
fn click_empty_viewport(world: &mut ApplicationWorld) {
    let [left, top, right, bottom] = world.viewport_bounds().map(|axis| axis as i32);
    let x = (right - left) / 8;
    let y = (bottom - top) / 8;
    world.click([
        1000.clamp(left + x, right - x),
        600.clamp(top + y, bottom - y),
    ]);
}

#[then("the Tools tab remains visible")]
fn tools_remains_visible(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    world.expect_no_text("Document Settings");
}

#[given("the ribbon is in tabs-only display with View active")]
fn view_in_tabs_only(world: &mut ApplicationWorld) {
    world.click_text("View");
    choose_tabs_mode(world);
    world.expect_no_text("User Interface");
}

#[given("the temporary View command groups are open")]
fn open_temporary_view(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text("User Interface");
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
}

#[then("the temporary Tools command groups replace the View command groups")]
fn tools_replace_view(world: &mut ApplicationWorld) {
    temporary_tools(world);
    world.expect_no_text("User Interface");
}

#[then("the temporary Tools command groups close")]
fn tools_commands_close(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_no_text("Document Settings");
}

#[then("Tools remains the active tab in tabs-only display")]
fn tools_selected(world: &mut ApplicationWorld) {
    let tools = world.pixel_left_of("Tools");
    let view = world.pixel_left_of("View");
    assert_ne!(tools, view, "expected Tools to be selected");
    world.expect_no_text("Document Settings");
}
