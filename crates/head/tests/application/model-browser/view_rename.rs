// SPDX-License-Identifier: GPL-2.0-only
//! Exercise part-view inline naming and duplicate correction in the native browser (#4118).
use super::part_views::{activate_copy_for_camera, expect_branch_label, select_all_text};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::contract::PartDesignView as _;

#[given("an active copied view is named \"Capture View\"")]
fn active_copy_named_capture(world: &mut ApplicationWorld) {
    activate_copy_for_camera(world);
    expect_active_name(world, "Capture View");
}
#[given("a sibling view is named \"Front\"")]
fn sibling_front(world: &mut ApplicationWorld) {
    world.expect_text("Front");
}
#[when("the user selects the copied view and presses F2")]
fn start_view_rename(world: &mut ApplicationWorld) {
    world.click_text("Capture View");
    world.key(oblikovati_viewport::ViewportPhysicalKey::F2, false);
    world.pump(Vec::new());
    assert!(world.text_is_selected("Capture View"));
}
#[when(expr = "the user replaces its name with {string} and presses Enter")]
#[when(expr = "the user replaces it with {string} and presses Enter")]
fn replace_view_name_and_accept(world: &mut ApplicationWorld, name: String) {
    replace_name(world, &name);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("the user replaces its name with \"Temporary View\"")]
fn replace_view_name_without_accept(world: &mut ApplicationWorld) {
    replace_name(world, "Temporary View");
}
fn replace_name(world: &mut ApplicationWorld, name: &str) {
    select_all_text(world);
    world.type_text(name);
    world.pump(Vec::new());
}
#[when("the user clears its selected name and presses Enter")]
fn clear_view_name(world: &mut ApplicationWorld) {
    select_all_text(world);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Backspace, false);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("a message says the new name is already used")]
fn duplicate_name_message(world: &mut ApplicationWorld) {
    assert!(
        world
            .text()
            .split_whitespace()
            .collect::<String>()
            .contains("Thenewnameisalreadyused.")
    );
}
#[then("the message offers Edit and cancels")]
fn duplicate_name_actions(world: &mut ApplicationWorld) {
    world.expect_text("Edit");
    world.expect_text("Cancel");
}
#[when("the user chooses Edit on the duplicate-name message")]
fn edit_duplicate_name(world: &mut ApplicationWorld) {
    world.click_text("Edit");
    world.pump(Vec::new());
}
#[then("the inline name field contains \"Front\"")]
fn duplicate_name_remains_editable(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("Front"));
}
#[then("the active view is named \"Capture View\"")]
#[then("the active view remains named \"Capture View\"")]
fn capture_name_remains(world: &mut ApplicationWorld) {
    expect_active_name(world, "Capture View");
}
#[then("the copied view is named \"Review View\"")]
fn review_name_applied(world: &mut ApplicationWorld) {
    expect_active_name(world, "Review View");
}
fn expect_active_name(world: &mut ApplicationWorld, name: &str) {
    let state = world.state();
    let views = state.browser.unwrap().part_design_views.unwrap();
    assert_eq!(views.active().unwrap().name(), name);
    world.expect_text(name);
}
#[then("the View branch identifies \"Capture View\" as active")]
fn capture_branch_active(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Capture View");
}
#[then("the View branch identifies \"Review View\" as active")]
fn review_branch_active(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Review View");
}
#[then("the View branch identifies \"front\" as active")]
fn lowercase_branch_active(world: &mut ApplicationWorld) {
    expect_branch_label(world, "front");
}
#[then("inline editing closes")]
fn view_inline_closed(world: &mut ApplicationWorld) {
    world.expect_text("Capture View");
    assert!(!world.text_is_selected("Capture View"));
}
#[then("no message is displayed")]
fn no_duplicate_name_message(world: &mut ApplicationWorld) {
    assert!(
        !world
            .text()
            .split_whitespace()
            .collect::<String>()
            .contains("Thenewnameisalreadyused.")
    );
}
#[then("both \"Front\" and \"front\" appear as sibling views")]
fn case_distinct_views(world: &mut ApplicationWorld) {
    world.expect_text("Front");
    world.expect_text("front");
}
#[then("the \"front\" view is checked")]
fn lowercase_view_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("front"));
}
