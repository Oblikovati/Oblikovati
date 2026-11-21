// SPDX-License-Identifier: GPL-2.0-only
//! Native viewport clicks resolve keyboard expansion retention (#4099).
use super::ApplicationWorld;
use oblikovati_viewport::ViewportPhysicalKey;

fn keyboard_open(world: &mut ApplicationWorld) {
    for key in [
        ViewportPhysicalKey::LeftAlt,
        ViewportPhysicalKey::M,
        ViewportPhysicalKey::Y,
    ] {
        world.key(key, false);
    }
}
fn face_click(world: &mut ApplicationWorld) {
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(world);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
fn expansion_visible() -> bool {
    super::diagnostics::window_geometry("##ribbon-additional-PartModify").is_some()
}
#[test]
fn x11_repeated_face_click_then_empty_dismisses_keyboard_expansion() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    keyboard_open(&mut world);
    let point = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    for _ in 0..2 {
        super::super::x11_input::click(point);
        world.pump(Vec::new());
        assert!(world.state().model_face_chosen);
        assert!(expansion_visible());
        assert!(world.frame.runs.iter().any(|run| run.text == "OD"));
    }
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    assert!(!world.state().model_face_chosen);
    assert!(!expansion_visible());
    assert!(!world.frame.runs.iter().any(|run| run.text == "OD"));
    world.close();
}
#[test]
fn x11_pinned_expansion_survives_face_click() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_pinned(&mut world);
    face_click(&mut world);
    assert!(world.state().model_face_chosen);
    assert!(expansion_visible());
    world.close();
}

#[test]
fn x11_navigation_consumed_click_cannot_leave_a_pending_pick() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::super::viewport_pan::press_bar(&mut world, "P");
    keyboard_open(&mut world);
    assert!(expansion_visible());
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    assert!(!expansion_visible());
    super::opened_unpinned(&mut world);
    super::super::x11_input::click([1000, 600]);
    world.pump(Vec::new());
    assert!(!expansion_visible());
    world.close();
}
