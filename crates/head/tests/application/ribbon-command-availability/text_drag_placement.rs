// SPDX-License-Identifier: GPL-2.0-only
//! Drive a Text placement through a native press, drag, and release.
use super::*;

#[test]
fn sketch_text_drag_placement_cancel_drops_the_pending_press() {
    let mut world = ApplicationWorld::new();
    start_text_from_part(&mut world);
    let point = [1000, 710];
    x11_input::move_pointer(point);
    x11_input::left_button(point, true);
    world.pump(Vec::new());
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    x11_input::left_button(point, false);
    world.pump(Vec::new());
    world.expect_ready();
    assert!(world.state().sketch_text.is_none());
    assert!(world.edited_geometry().texts().is_empty());
    world.close();
}

#[test]
fn sketch_text_drag_placement_waits_for_release_and_keeps_insertion_point() {
    let mut world = ApplicationWorld::new();
    start_text_from_part(&mut world);
    let from = [1050, 710];
    let to = [1250, 810];
    let anchor = world.sketch_point_of(from);
    drag_text_to(&mut world, from, to);
    world.expect_text("Format Text");
    assert!(world.state().sketch_text.unwrap().content.is_empty());
    assert!(world.edited_geometry().texts().is_empty());
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("format-text-after-drag");
    }
    type_content(&mut world, b'b');
    click_editor_button(&mut world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
    let sketch = world.edited_geometry();
    let [text] = sketch.texts() else {
        panic!("expected one accepted Text");
    };
    assert_eq!(text.style.content, "b");
    assert_eq!(sketch.point(text.anchor).unwrap().position, anchor);
    world.close();
}

fn drag_text_to(world: &mut ApplicationWorld, from: [i32; 2], to: [i32; 2]) {
    x11_input::move_pointer(from);
    x11_input::left_button(from, true);
    world.pump(Vec::new());
    assert!(world.state().sketch_text.is_none(), "Text opened on press");
    x11_input::move_pointer(to);
    world.pump(Vec::new());
    assert!(
        world.state().sketch_text.is_none(),
        "Text opened during drag"
    );
    x11_input::left_button(to, false);
    world.pump(Vec::new());
}
