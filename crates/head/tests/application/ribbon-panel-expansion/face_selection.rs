// SPDX-License-Identifier: GPL-2.0-only
//! Face selection while an additional ribbon area is open (#3856).
use super::ApplicationWorld;
use cucumber::{given, then, when};

#[when("the user clicks a visible solid face outside the expanded area")]
fn click_exposed_face(world: &mut ApplicationWorld) {
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(world);
    world.click(point);
    world.pump(Vec::new());
}
#[then("the face selection is cleared")]
fn face_selection_is_cleared(world: &mut ApplicationWorld) {
    assert!(
        !world.state().model_face_chosen,
        "expected no selected model face"
    );
}

#[given("the full 3D Model ribbon is displayed with the Modify area closed")]
fn full_model_without_expansion(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
}
#[then("the Modify additional command area remains visible with keyboard tips displayed")]
fn expansion_with_key_tips(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    assert!(
        world.frame.runs.iter().any(|run| run.text == "OD"),
        "expected the Move Bodies key tip"
    );
}
#[then("the keyboard tips and Modify additional command area close")]
fn expansion_and_key_tips_closed(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(!world.frame.runs.iter().any(|run| run.text == "OD"));
}
#[then("the clicked face is visibly selected")]
fn selected_face_is_visible(world: &mut ApplicationWorld) {
    assert!(
        world.state().model_face_chosen,
        "expected a selected model face"
    );
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_selected_face_fill(&capture, &windows),
        "expected selected face pixels outside GUI windows"
    );
}

#[cfg(target_os = "linux")]
#[test]
fn x11_face_click_selects_the_solid_and_closes_unpinned_expansion() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen);
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    face_selection_is_cleared(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_face_click_keeps_keyboard_opened_expansion_until_escape() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    for key in [
        ViewportPhysicalKey::LeftAlt,
        ViewportPhysicalKey::M,
        ViewportPhysicalKey::Y,
    ] {
        world.key(key, false);
    }
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    expansion_with_key_tips(&mut world);
    assert!(world.state().model_face_chosen);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    expansion_and_key_tips_closed(&mut world);
    selected_face_is_visible(&mut world);
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    face_selection_is_cleared(&mut world);
    world.close();
}

#[when("the user right-clicks a visible solid face outside the expanded area")]
fn right_click_exposed_face(world: &mut ApplicationWorld) {
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(world);
    world.right_click(point);
    world.pump(Vec::new());
}
#[then("the face context menu is displayed")]
fn face_context_menu_visible(world: &mut ApplicationWorld) {
    assert!(
        world.state().model_face_chosen,
        "expected the context face to be selected"
    );
    world.expect_text("New Sketch");
    world.expect_text("Properties");
}
#[then("the face context menu closes")]
fn face_context_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Properties");
}

#[cfg(target_os = "linux")]
#[test]
fn x11_face_context_click_selects_face_and_escape_preserves_it() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    super::super::x11_input::right_click(point);
    world.pump(Vec::new());
    face_context_menu_visible(&mut world);
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    face_context_menu_closed(&mut world);
    assert!(super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(world.state().model_face_chosen);
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    face_selection_is_cleared(&mut world);
    world.close();
}
