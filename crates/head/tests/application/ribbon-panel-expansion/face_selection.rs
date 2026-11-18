// SPDX-License-Identifier: GPL-2.0-only
//! Face selection while an additional ribbon area is open (#3856).
use super::ApplicationWorld;
use cucumber::{then, when};

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
