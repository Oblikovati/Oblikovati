// SPDX-License-Identifier: GPL-2.0-only
//! Verify the ribbon when the user switches between a part and Home.
use super::*;

#[given("the Home workspace is available as a document tab")]
fn home_tab_available(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
}
#[then("the Home workspace is displayed")]
fn home_workspace_visible(world: &mut ApplicationWorld) {
    assert!(window_geometry("##home-workspace").is_some());
    world.expect_text("Recent");
}
#[then("the full Tools ribbon remains displayed with Home-context commands")]
#[then("the full Tools ribbon is displayed with part-context commands")]
fn tools_commands_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("Document Settings is unavailable")]
fn document_settings_disabled(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Document Settings"));
}
#[then("Document Settings is available")]
fn document_settings_enabled(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Document Settings"));
}
#[when("the user selects the open part document tab")]
fn activate_open_part(world: &mut ApplicationWorld) {
    world.click_tab(world.previous_document.expect("expected an open part"));
}
#[then("the part workspace and its model browser are displayed")]
#[then("the part workspace and model browser appear")]
fn part_workspace_visible(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected an open part");
    world.expect_active(part);
    world.expect_browser_of(part);
    assert!(window_geometry("##home-workspace").is_none());
}

#[given("the ribbon display policy is \"Minimize to Tabs\"")]
fn tabs_policy_selected(world: &mut ApplicationWorld) {
    click_display_menu(world);
    if world.checked("Minimize to Tabs") {
        world.escape();
    } else {
        world.click_text("Minimize to Tabs");
        activate_display_control(world);
    }
}
#[given("the ribbon is minimized to tabs")]
fn minimize_to_tabs(world: &mut ApplicationWorld) {
    choose_tabs_mode(world);
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[given("the Tools ribbon tab is visible")]
#[then("the Tools ribbon tab is visible")]
fn tools_tab_visible(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
}
#[then("the Tools ribbon tab remains selected")]
#[then("the Tools ribbon tab is selected")]
fn tools_tab_selected(world: &mut ApplicationWorld) {
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("View");
    assert_eq!(world.pixel_left_of("Manage"), inactive);
    assert_ne!(world.pixel_left_of("Tools"), inactive);
}

#[given("the Home workspace is active")]
fn activate_home(world: &mut ApplicationWorld) {
    world.click(super::super::diagnostics::pointer_pixels(
        world.expect_tab("Home").center(),
    ));
    home_workspace_visible(world);
}
#[then("the Home-context Tools commands appear temporarily")]
#[then("the part-context Tools commands appear temporarily")]
fn temporary_tools_commands(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}
#[then("the temporary Home-context commands are closed")]
#[then("the temporary part-context commands are closed")]
fn temporary_commands_closed(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_no_text("Application Options");
}
#[then("the part ribbon remains minimized to tabs")]
#[then("the Home ribbon remains minimized to tabs")]
fn tabs_only_remains(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    temporary_commands_closed(world);
}
