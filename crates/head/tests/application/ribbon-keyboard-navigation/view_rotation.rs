// SPDX-License-Identifier: GPL-2.0-only
//! View rotation while the ribbon key tips are visible.
use super::*;
use crate::HeadWindow;

#[given("the full 3D Model ribbon is selected")]
fn model_selected(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
}
#[when("the user presses Alt to show top-level ribbon key tips")]
fn show_tips(world: &mut ApplicationWorld) {
    world.observed_view_camera = Some(super::super::viewport_pan::camera(world));
    super::alt(world);
}
#[when("the user presses the Right Arrow key")]
fn rotate_right(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::RightArrow, false);
}
#[then("the displayed solid changes orientation")]
fn orientation_changed(world: &mut ApplicationWorld) {
    let before = world.observed_view_camera.unwrap();
    let after = world.app().window.view_camera().unwrap();
    assert_ne!(before.eye, after.eye);
    super::super::viewport_orientation::solid_extent(world);
}
#[when("the user presses the Left Arrow key")]
fn rotate_left(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::LeftArrow, false);
}
#[then("the solid returns to its prior visible orientation")]
fn orientation_restored(world: &mut ApplicationWorld) {
    let before = world.observed_view_camera.unwrap();
    let after = world.app().window.view_camera().unwrap();
    for (old, new) in before
        .eye
        .into_iter()
        .chain(before.up)
        .zip(after.eye.into_iter().chain(after.up))
    {
        assert!(
            (old - new).abs() < 1e-6,
            "orientation changed from {before:?} to {after:?}"
        );
    }
    super::super::viewport_orientation::solid_extent(world);
}
#[when("the user presses the 3D Model tab key tip")]
fn model_tip(world: &mut ApplicationWorld) {
    super::model_key(world);
}
#[then("the ribbon shows command-level key tips")]
fn command_tips(world: &mut ApplicationWorld) {
    super::command_tips(world);
}

#[test]
fn browser_arrow_moves_selection_without_rotating_view() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("YZ Plane");
    let before = world.app().window.view_camera().unwrap();
    world.key(ViewportPhysicalKey::DownArrow, false);
    let after = world.app().window.view_camera().unwrap();
    assert_eq!(before, after, "browser navigation must preserve the view");
    super::alt(&mut world);
    world.key(ViewportPhysicalKey::UpArrow, false);
    assert_eq!(before, world.app().window.view_camera().unwrap());
    assert!(super::exact_tip(&world, "M"));
    world.close();
}
