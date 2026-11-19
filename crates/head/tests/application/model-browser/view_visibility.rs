// SPDX-License-Identifier: GPL-2.0-only
//! Switch locked and unlocked views over a rendered solid (#4118).
use super::part_views::{
    activate_copied_view as activate_copy_row, copied_view_active, copy_primary_view,
    expanded_primary_view, rename_copied_view,
};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::contract::PartDesignView as _;

#[given("The MCAD Software has a part containing one visible solid body")]
fn visible_prism(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    expect_solid_visible(world);
}
#[given("an unlocked copied view is active")]
fn unlocked_copy_active(world: &mut ApplicationWorld) {
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_copied_view(world);
    activate_copy_row(world);
    assert!(!active_view_locked(world));
}
#[given("the solid body is visible in the Primary view")]
fn primary_saved_visible(world: &mut ApplicationWorld) {
    let state = world.state();
    let views = state.browser.unwrap().part_design_views.unwrap();
    assert!(views.get(1).unwrap().hidden_body_keys().is_empty());
    expect_solid_visible(world);
}
#[when("the user chooses Lock from the copied view context menu")]
fn lock_copy(world: &mut ApplicationWorld) {
    choose_copy_action(world, "Lock");
}
#[when("the user chooses All Hidden from the copied view context menu")]
fn hide_copy_bodies(world: &mut ApplicationWorld) {
    choose_copy_action(world, "All Hidden");
}
fn choose_copy_action(world: &mut ApplicationWorld, label: &str) {
    right_click_row(world, "Capture View");
    choose_menu_row(world, label);
    world.pump(Vec::new());
}
#[then("the solid body is hidden in the viewport")]
fn prism_hidden(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let status = world
        .app()
        .window
        .native
        .scene_status()
        .expect("expected a published scene");
    assert_eq!(status.visible_bodies, 0);
    assert!(!super::super::solid_visibility::visible_solid_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}
#[then("the copied view displays a lock indicator")]
fn copied_view_is_locked(world: &mut ApplicationWorld) {
    assert!(active_view_locked(world));
    assert!(lock_icon_visible(world));
}
fn active_view_locked(world: &mut ApplicationWorld) -> bool {
    world
        .state()
        .browser
        .unwrap()
        .part_design_views
        .unwrap()
        .active()
        .unwrap()
        .locked()
}
fn lock_icon_visible(world: &mut ApplicationWorld) -> bool {
    let row = world.run("Capture View").clone();
    let left = row.maximum[0] as usize + 5;
    let top = row.minimum[1] as usize;
    (top..top + 15).any(|y| {
        (left..left + 12).any(|x| {
            let color = world.pixel_at([x, y]);
            color[0] > 150 && color[1] > 150 && color[2] > 150
        })
    })
}
#[when("the user activates the Primary view")]
fn activate_primary_view(world: &mut ApplicationWorld) {
    world.double_click(diagnostics::pointer_pixels(world.run("Primary").center()));
    world.pump(Vec::new());
}
#[when("the user activates the locked copied view")]
#[when("the user activates the copied view again")]
fn activate_copied_view(world: &mut ApplicationWorld) {
    activate_copy_row(world);
}
#[then("the copied view displays its active and locked indicators")]
fn active_copy_is_locked(world: &mut ApplicationWorld) {
    copied_view_active(world);
    copied_view_is_locked(world);
}
#[then("the copied view is active")]
fn copy_is_active(world: &mut ApplicationWorld) {
    copied_view_active(world);
}

#[then("the solid body is visible")]
fn visible_prism_after_switch(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
    let status = world
        .app()
        .window
        .native
        .scene_status()
        .expect("expected a published scene");
    assert_eq!(status.visible_bodies, 1);
}
