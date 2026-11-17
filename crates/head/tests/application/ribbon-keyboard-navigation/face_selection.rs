// SPDX-License-Identifier: GPL-2.0-only
//! Face selection while the ribbon key tips are open.
use super::*;
#[cfg(target_os = "linux")]
use crate::HeadWindow;

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

#[cfg(target_os = "linux")]
#[test]
fn native_face_click_cue_retires_when_pointer_leaves() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    world.click_text("Select");
    world.click_text("Faces");
    super::super::x11_input::focus_window();
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    let (low, high) = super::super::solid_visibility::visible_fixture_bounds(&capture, &windows)
        .expect("expected a visible solid face outside the UI");
    let at = [
        ((low[0] + high[0]) / 2) as i32,
        ((low[1] + high[1]) / 2) as i32,
    ];
    super::super::x11_input::move_pointer(at);
    world.pump(Vec::new());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
    assert!(
        world.state().model_face_chosen,
        "native click at {at:?}: expected a selected face; text: {}",
        world.text()
    );
    toolbar_visible(&mut world);
    let cue = world.state().model_pick_cue.clone();
    assert!(cue.is_some(), "selected face should retain its model cue");
    let projected = world
        .app()
        .window
        .model_pixel(cue.unwrap().position)
        .unwrap();
    assert!(
        (projected[0] - at[0]).abs() <= 12 && (projected[1] - at[1]).abs() <= 12,
        "cue projects to {projected:?}, click is {at:?}"
    );
    world.expect_text("1. Face");
    super::super::x11_input::move_pointer([at[0] + 40, at[1] + 40]);
    world.pump(Vec::new());
    world.expect_no_text("1. Face");
    assert!(
        world.state().model_face_chosen,
        "pointer movement should retain the face"
    );
    super::super::x11_input::move_pointer(at);
    world.pump(Vec::new());
    world.expect_no_text("1. Face");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_default_filter_selects_and_clears_visible_face() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let (low, high) = super::super::viewport_orientation::solid_bounds(&mut world);
    let point = [
        ((low[0] + high[0]) / 2) as i32,
        ((low[1] + high[1]) / 2) as i32,
    ];
    let ray = world.app().window.view_line(point).unwrap();
    let edges = world.app().owner.face_edges_on_line(ray).unwrap();
    assert!(
        !edges.is_empty(),
        "pixel {point:?}: expected an analytic face hit"
    );
    super::super::x11_input::focus_window();
    super::super::x11_input::move_pointer(point);
    world.pump(Vec::new());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert!(
        world.state().model_face_chosen,
        "default face click {point:?}: {}",
        world.text()
    );
    assert_eq!(
        world.app().owner.selected_browser_outline(),
        edges,
        "selected face must match the analytic hit"
    );
    world.expect_text("1. Face");
    toolbar_visible(&mut world);
    let empty = super::super::viewport_pan::EMPTY_VIEW;
    super::super::x11_input::move_pointer(empty);
    world.pump(Vec::new());
    super::super::x11_input::click(empty);
    world.pump(Vec::new());
    selection_cleared(&mut world);
    world.expect_no_text("1. Face");
    world.close();
}
