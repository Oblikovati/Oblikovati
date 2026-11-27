// SPDX-License-Identifier: GPL-2.0-only
//! Face selection while the ribbon key tips are open.
use super::*;
#[cfg(target_os = "linux")]
use crate::HeadWindow;

#[given("no model face is selected")]
fn no_face_selected(world: &mut ApplicationWorld) {
    assert!(!world.state().model_face_chosen);
}

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
    let at = visible_face_point(world);
    world.click(at);
    world.pump(Vec::new());
}
pub(crate) fn visible_face_point(world: &mut ApplicationWorld) -> [i32; 2] {
    use crate::HeadWork;
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    let (low, high) = super::super::solid_visibility::visible_fixture_bounds(&capture, &windows)
        .expect("expected a visible solid face outside the UI");
    let at = [
        ((low[0] + high[0]) / 2) as i32,
        ((low[1] + high[1]) / 2) as i32,
    ];
    let ray = world.app().window.view_line(at).unwrap();
    let hit = world.app().owner.face_edges_on_line(ray).unwrap();
    assert!(
        !hit.is_empty(),
        "point {at:?}: expected an analytic face hit"
    );
    at
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
#[cfg(target_os = "linux")]
#[test]
fn native_face_click_dismisses_key_tips_and_selects_face() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    show_commands(&mut world);
    commands_visible(&mut world);
    let at = visible_face_point(&mut world);
    super::super::x11_input::click(at);
    world.pump(Vec::new());
    tips_disappear(&mut world);
    face_selected(&mut world);
    toolbar_visible(&mut world);
    world.save_capture("keytips-native-face-3856");
    world.close();
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
    let toolbar_before = super::super::diagnostics::window_geometry("##selection-toolbar");
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
    world.run_in_window("##status-bar", "1Face");
    super::super::x11_input::move_pointer([at[0] + 40, at[1] + 40]);
    world.pump(Vec::new());
    world.expect_no_text("1. Face");
    assert_eq!(
        super::super::diagnostics::window_geometry("##selection-toolbar"),
        toolbar_before
    );
    assert!(
        world.state().model_face_chosen,
        "pointer movement should retain the face"
    );
    super::super::x11_input::move_pointer(at);
    world.pump(Vec::new());
    world.expect_no_text("1. Face");
    world.close();
}

#[given("a part is active in the MCAD Software with one face selected")]
fn part_with_selected_face(world: &mut ApplicationWorld) {
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(world);
    click_face(world);
    face_selected(world);
}

#[given("one visible solid face is selected")]
fn select_visible_face(world: &mut ApplicationWorld) {
    click_face(world);
    one_face_still_selected(world);
}

#[then("key tips appear on the Tools ribbon commands")]
fn tools_command_tips(world: &mut ApplicationWorld) {
    assert!(super::exact_tip(world, "OA"));
    assert!(super::exact_tip(world, "OD"));
}
#[then("the top-level and Tools command key tips remain visible")]
fn selected_face_tips_remain(world: &mut ApplicationWorld) {
    super::tips_shown(world);
    tools_command_tips(world);
    face_selected(world);
}
#[then("the selection indicator still reports one face")]
fn one_face_still_selected(world: &mut ApplicationWorld) {
    face_selected(world);
    assert_eq!(world.state().selected_face_appearances.len(), 1);
    world.run_in_window("##status-bar", "1Face");
}

#[cfg(target_os = "linux")]
#[test]
fn native_selected_face_tools_tips_survive_escape_and_open_options() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let face = visible_face_point(&mut world);
    super::super::x11_input::click(face);
    world.pump(Vec::new());
    face_selected(&mut world);
    let tools = world.run("Tools").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(tools));
    world.pump(Vec::new());
    for key in [0xffe9, 0xff1b] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    selected_face_tips_remain(&mut world);
    one_face_still_selected(&mut world);
    for key in [u32::from(b'o'), u32::from(b'a')] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    assert!(super::super::diagnostics::window_geometry("Application Options").is_some());
    face_selected(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_default_filter_selects_and_clears_visible_face() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    assert_eq!(
        world.app().owner.face_mesh_builds(),
        1,
        "opening the part must prepare the named display mesh"
    );
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
    let before = super::super::capture_framebuffer(&mut world.app().window.native);
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
    assert_eq!(
        world.app().owner.face_mesh_builds(),
        1,
        "selecting a face must reuse the display mesh"
    );
    let selected = super::super::capture_framebuffer(&mut world.app().window.native);
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("selected-named-face-4048");
    }
    let color = capture_pixel(&selected, point);
    assert!(
        color[1] > color[0] + 20 && color[2] > color[0] + 20,
        "selected face pixel {color:?}: expected teal"
    );
    assert!(
        outline_pixels(&selected) > outline_pixels(&before) + 10,
        "selected face: expected the teal selection boundary"
    );
    let empty = super::super::viewport_pan::EMPTY_VIEW;
    super::super::x11_input::move_pointer(empty);
    world.pump(Vec::new());
    super::super::x11_input::click(empty);
    world.pump(Vec::new());
    selection_cleared(&mut world);
    assert!(
        !world.frame.runs.iter().any(|run| run.text == "1 Face"),
        "the status bar must clear its face count"
    );
    assert_eq!(
        world.app().owner.face_mesh_builds(),
        1,
        "clearing selection must reuse the display mesh"
    );
    world.expect_no_text("1. Face");
    let cleared = super::super::capture_framebuffer(&mut world.app().window.native);
    assert_eq!(
        capture_pixel(&cleared, point),
        capture_pixel(&before, point),
        "cleared face must recover its original appearance"
    );
    assert!(
        outline_pixels(&cleared) <= outline_pixels(&before) + 10,
        "cleared face: expected the teal selection boundary to disappear"
    );
    world.close();
}

#[cfg(target_os = "linux")]
fn outline_pixels(capture: &oblikovati_viewport::ViewportCapture) -> usize {
    const DEFAULT_DARK_FACE_OUTLINE: [u8; 3] = [94, 234, 212];
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| pixel[..3] == DEFAULT_DARK_FACE_OUTLINE)
        .count()
}

#[cfg(target_os = "linux")]
fn capture_pixel(capture: &oblikovati_viewport::ViewportCapture, point: [i32; 2]) -> [u8; 4] {
    let index = (point[1] as usize * capture.extent[0] as usize + point[0] as usize) * 4;
    capture.rgba[index..index + 4].try_into().unwrap()
}
