// SPDX-License-Identifier: GPL-2.0-only
//! The accepted Part preference controls sketch creation and edit framing.
use super::super::{ApplicationWorld, diagnostics, viewport_orientation};
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;

fn set_look_at(world: &mut ApplicationWorld, enabled: bool) {
    world.click_text("Tools");
    world.click_text("Application Options");
    assert_eq!(world.checked("Look at sketch plane"), !enabled);
    world.click_text("Look at sketch plane");
    world.click_text("Apply Part");
    assert_eq!(
        world.state().part_options.unwrap().look_at_sketch_plane,
        enabled
    );
    world.escape();
    world.click_text("3D Model");
}
#[given("Part6 is in an oblique Home View")]
fn part_in_home_view(world: &mut ApplicationWorld) {
    for _ in 0..6 {
        world.add_part();
    }
    world.add_finished_sketch();
    world.expect_text("Part6");
    viewport_orientation::press_part(world, |layout| layout.home);
    let camera = world.app().window.native.view.camera_state().unwrap();
    assert!(camera.eye.iter().all(|axis| *axis != 0.));
    world.observed_view_camera = Some(camera);
}
#[when("the user disable Part Look at sketch plane and create a sketch on YZ Plane")]
fn create_without_framing(world: &mut ApplicationWorld) {
    set_look_at(world, false);
    world.click_text("Start 2D Sketch");
    world.click_text("YZ Plane");
    world.pump(Vec::new());
}
#[then("Sketch2 is created and the oblique view remains")]
fn sketch_keeps_home_view(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 2);
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    assert_eq!(
        world.app().window.native.view.camera_state().unwrap(),
        world.observed_view_camera.unwrap()
    );
}
#[when("the user finish Sketch2 and restore Part Look at sketch plane")]
fn finish_and_restore_framing(world: &mut ApplicationWorld) {
    world.finish_sketch();
    set_look_at(world, true);
}
#[when("the user choose Edit Sketch on that same Sketch2")]
fn edit_same_sketch(world: &mut ApplicationWorld) {
    let row = world.run("Sketch2").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
}
#[then("its editing environment opens in RIGHT view")]
fn edited_sketch_faces_right(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    viewport_orientation::expect_control_shows(world, "RIGHT");
    let camera = world.app().window.native.view.camera_state().unwrap();
    assert!(camera.eye[0] > camera.target[0]);
    assert_eq!(camera.eye[1..], camera.target[1..]);
    let sketch = world.state().browser.unwrap().sketches[1].id;
    assert_eq!(world.state().active_sketch, Some(sketch));
}
