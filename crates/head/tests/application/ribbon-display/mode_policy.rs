// SPDX-License-Identifier: GPL-2.0-only
//! Verify ribbon minimization policy and visible Tools groups.
use super::*;

#[given("the Tools ribbon is expanded")]
fn tools_expanded(world: &mut ApplicationWorld) {
    full_tools(world);
    world.ribbon_height = band_height();
}

#[given("Minimize to Tabs is the selected ribbon display policy")]
fn tabs_policy_with_full_ribbon(world: &mut ApplicationWorld) {
    click_display_menu(world);
    if world.checked("Minimize to Tabs") {
        world.escape();
    } else {
        world.click_text("Minimize to Tabs");
        activate_display_control(world);
    }
    full_tools(world);
}

#[when("the user opens the ribbon display policy menu")]
fn open_policy_menu(world: &mut ApplicationWorld) {
    click_display_menu(world);
}

#[then("Minimize to Tabs is marked as selected")]
fn tabs_policy_selected(world: &mut ApplicationWorld) {
    tabs_checked(world);
}

#[then("Minimize to Panel Titles, Minimize to Panel Buttons and Cycle Through All are listed")]
fn other_policies_listed(world: &mut ApplicationWorld) {
    menu_choices(world);
    cycle_choice(world);
}

#[then("the full Tools ribbon remains visible")]
fn full_tools_remains(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}

#[then("the menu closes without changing the display policy")]
fn policy_remains_selected(world: &mut ApplicationWorld) {
    world.expect_no_text("Minimize to Tabs");
    click_display_menu(world);
    tabs_checked(world);
    world.escape();
}

#[then("the Tools command groups are reduced to a narrow strip of titles")]
#[then("the Tools ribbon displays the group-title strip")]
fn tools_title_strip(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    world.expect_no_text("Application Options");
    expect_shorter_band(world);
}

#[then("the Tools command groups appear as compact buttons")]
fn tools_compact_buttons(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    world.expect_no_text("Application Options");
    expect_shorter_band(world);
}

#[then("the display menu closes and the group-title strip remains")]
#[then("the display menu closes and the compact buttons remain")]
fn compact_tools_remains(world: &mut ApplicationWorld) {
    world.expect_no_text("Minimize to Tabs");
    world.expect_text("Options");
    world.expect_no_text("Application Options");
}
