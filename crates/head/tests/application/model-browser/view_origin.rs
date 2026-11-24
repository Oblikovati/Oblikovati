// SPDX-License-Identifier: GPL-2.0-only
//! Origin plane visibility remains independent of part views (#4118).
use super::part_views::activate_copy_for_camera;
use super::view_fixture::{expect_prepared_copy, right_click_copy};
use super::view_visibility::{active_view_locked, copied_view_is_locked, lock_icon_visible};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::SetPartDesignViewOptionsArgs};

#[given("The MCAD Software has an empty part with an unlocked copied view active")]
fn unlocked_copy_with_oblique_camera(world: &mut ApplicationWorld) {
    if expect_prepared_copy(world) {
        assert!(!active_view_locked(world));
        return;
    }
    world.add_part();
    orient_oblique(world);
    activate_copy_for_camera(world);
}
#[given("The MCAD Software has an empty part with a locked copied view active")]
fn locked_copy_with_oblique_camera(world: &mut ApplicationWorld) {
    if expect_prepared_copy(world) {
        assert!(active_view_locked(world));
        return;
    }
    unlocked_copy_with_oblique_camera(world);
    choose_view_lock(world);
}
#[given("the Primary view is locked")]
fn primary_locked_fixture(world: &mut ApplicationWorld) {
    Client::new(world.app().owner.document_host())
        .part_design_views()
        .set_options(&SetPartDesignViewOptionsArgs {
            id: 1,
            locked: Some(Box::new(true)),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}
#[given("the YZ origin plane is hidden")]
fn yz_plane_initially_hidden(world: &mut ApplicationWorld) {
    orient_oblique(world);
    world.expect_text("YZ Plane");
    assert!(!plane_shows_at(world, "YZ Plane"));
}
fn orient_oblique(world: &mut ApplicationWorld) {
    super::super::viewport_orientation::press_part(world, |parts| {
        [
            parts.center[0] + parts.half * 0.85,
            parts.center[1] - parts.half * 0.85,
        ]
    });
}
#[when("the user chooses Lock from the active view context menu")]
#[when("the user chooses Lock from the active view context menu again")]
pub(super) fn choose_view_lock(world: &mut ApplicationWorld) {
    right_click_copy(world);
    choose_menu_row(world, "Lock");
    world.pump(Vec::new());
}
#[then("the active view displays a lock indicator")]
#[then("the active view still displays a lock indicator")]
fn active_view_lock_visible(world: &mut ApplicationWorld) {
    copied_view_is_locked(world);
}
#[then("the active view no longer displays a lock indicator")]
fn active_view_lock_absent(world: &mut ApplicationWorld) {
    assert!(!active_view_locked(world));
    assert!(!lock_icon_visible(world));
}
#[when("the user chooses Visibility from the YZ origin plane context menu")]
#[when("the user chooses Visibility from the YZ origin plane context menu again")]
fn toggle_yz_plane(world: &mut ApplicationWorld) {
    if !world
        .frame
        .runs
        .iter()
        .any(|row| row.text.trim() == "Visibility")
    {
        right_click_row(world, "YZ Plane");
    }
    choose_menu_row(world, "Visibility");
    click_empty_viewport(world);
    world.pump(Vec::new());
}
#[then("the YZ origin plane is visible")]
#[then("the YZ origin plane remains visible")]
fn yz_plane_visible(world: &mut ApplicationWorld) {
    assert!(plane_shows_at(world, "YZ Plane"));
}
#[then("the YZ origin plane is hidden")]
#[then("the YZ origin plane remains hidden")]
fn yz_plane_hidden(world: &mut ApplicationWorld) {
    assert!(!plane_shows_at(world, "YZ Plane"));
}
#[then("Visibility is checked in the plane context menu")]
fn yz_visibility_checked(world: &mut ApplicationWorld) {
    right_click_row(world, "YZ Plane");
    assert!(world.checked("Visibility"));
}
