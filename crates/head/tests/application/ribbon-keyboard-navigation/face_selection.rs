// SPDX-License-Identifier: GPL-2.0-only
//! Face selection while the ribbon key tips are open.
use super::*;

#[when("the user presses Alt and then the 3D Model tab key tip")]
fn show_commands(world: &mut ApplicationWorld) {
    super::show_model_tips(world);
}
#[then("command-level ribbon key tips are visible")]
fn commands_visible(world: &mut ApplicationWorld) {
    super::command_tips(world);
}
#[when("the user clicks a visible solid face")]
fn click_face(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    let at = super::super::solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected a visible solid face outside the UI");
    world.click(at);
    world.pump(Vec::new());
}
#[then("the ribbon key tips disappear")]
fn tips_disappear(world: &mut ApplicationWorld) {
    assert!(!super::exact_tip(world, "HO"));
    assert!(!super::exact_tip(world, "M"));
}
#[then("the clicked face is selected")]
fn face_selected(world: &mut ApplicationWorld) {
    assert!(world.state().model_face_chosen);
}
#[then("a contextual mini toolbar appears beside the selected face")]
fn toolbar_visible(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("##selection-toolbar").is_some());
    world.run_in_window("##selection-toolbar", "ZoomSelected");
}
#[then("the face selection and contextual mini toolbar clear")]
fn selection_cleared(world: &mut ApplicationWorld) {
    assert!(!world.state().model_face_chosen);
    assert!(super::super::diagnostics::window_geometry("##selection-toolbar").is_none());
}
#[test]
fn face_toolbar_zoom_frames_the_selected_face() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    click_face(&mut world);
    face_selected(&mut world);
    toolbar_visible(&mut world);
    let before = super::super::viewport_pan::camera(&mut world);
    let at = world
        .run_in_window("##selection-toolbar", "ZoomSelected")
        .center();
    world.click(at.map(|value| value as i32));
    let after = super::super::viewport_pan::camera(&mut world);
    assert!(
        after.scale < before.scale,
        "expected the selected face to fill more of the view"
    );
    world.close();
}

#[test]
fn choosing_a_feature_replaces_face_selection_and_its_toolbar() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    click_face(&mut world);
    face_selected(&mut world);
    toolbar_visible(&mut world);
    world.click_text("Extrude1");
    assert!(!world.state().model_face_chosen);
    world.run_in_window("##selection-toolbar", "EditSketch");
    world.close();
}
