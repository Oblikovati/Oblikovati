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
fn click_empty_viewport(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
}

#[then("the Tools tab remains visible")]
fn tools_remains_visible(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    world.expect_no_text("Document Settings");
}
