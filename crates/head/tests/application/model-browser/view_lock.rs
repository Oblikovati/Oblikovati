// SPDX-License-Identifier: GPL-2.0-only
//! Verify an inactive copied view's lock and context menu (#4118).
use super::part_views::inactive_view_copy;
use super::view_visibility::lock_icon_visible;
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::contract::PartDesignView as _;

#[given("an unlocked copied view representation exists in the expanded View branch")]
fn inactive_unlocked_copy(world: &mut ApplicationWorld) {
    inactive_view_copy(world);
    assert!(!copied_view_locked(world));
}
#[when("the user opens the copied view representation's context menu")]
#[when("the user reopens the copied view representation's context menu")]
#[when("the user reopens its context menu")]
fn copied_view_menu(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
}
#[then("Lock is unchecked")]
fn lock_menu_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Lock"));
}
#[then("Lock is checked")]
fn lock_menu_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Lock"));
}
#[then("Publish is checked")]
fn publish_menu_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Publish"));
}
#[then("activates and Copy are available")]
fn activate_and_copy_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Activate"));
    assert!(world.run_is_enabled("Copy"));
}
#[then("Edit View is unavailable")]
fn edit_view_unavailable(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Edit View"));
}
#[when("the user chooses Lock")]
#[when("the user chooses Lock again")]
fn choose_lock(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Lock");
}
#[then("a lock symbol appears beside the copied view representation")]
fn inactive_copy_is_locked(world: &mut ApplicationWorld) {
    assert!(copied_view_locked(world));
    assert!(lock_icon_visible(world));
}
#[then("the lock symbol disappears from the copied view representation")]
fn inactive_copy_is_unlocked(world: &mut ApplicationWorld) {
    assert!(!copied_view_locked(world));
    assert!(!lock_icon_visible(world));
}
fn copied_view_locked(world: &mut ApplicationWorld) -> bool {
    let state = world.state();
    let views = state.browser.unwrap().part_design_views.unwrap();
    views
        .list()
        .iter()
        .find(|view| view.name() == "Capture View")
        .unwrap()
        .locked()
}

#[test]
fn escape_closes_the_inactive_view_menu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    inactive_unlocked_copy(&mut world);
    copied_view_menu(&mut world);
    world.expect_text("Lock");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
    world.expect_no_text("Lock");
    world.expect_no_text("Edit View");
}
