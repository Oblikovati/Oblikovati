// SPDX-License-Identifier: GPL-2.0-only
//! Window-manager behavior for the application-owned frame (#4063).
use super::{ApplicationWorld, x11_input};
use std::time::{Duration, Instant};

fn pump_until(world: &mut ApplicationWorld, expected: bool) {
    let deadline = Instant::now() + Duration::from_secs(5);
    loop {
        world.pump(Vec::new());
        if world.app().window.native.view.window_maximized().unwrap() == expected {
            break;
        }
        assert!(Instant::now() < deadline, "expected maximized={expected}");
        std::thread::sleep(Duration::from_millis(30));
    }
}
fn press_maximize(world: &mut ApplicationWorld) {
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    x11_input::click([width - 72, 16]);
    world.pump(Vec::new());
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_maximizes_and_restores_under_a_window_manager() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    let original = world.app().window.native.drawable_extent().unwrap();
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    press_maximize(&mut world);
    pump_until(&mut world, true);
    let maximized = world.app().window.native.drawable_extent().unwrap();
    assert!(maximized[0] > original[0] && maximized[1] > original[1]);
    press_maximize(&mut world);
    pump_until(&mut world, false);
    assert_eq!(
        world.app().window.native.drawable_extent().unwrap(),
        original
    );
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_resizes_from_bottom_right_and_keeps_opposite_corner() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    let before = x11_input::application_bounds();
    let edge = [before[2] - 2, before[3] - 2];
    x11_input::move_pointer(edge);
    world.pump(Vec::new());
    x11_input::left_button(edge, true);
    world.pump(Vec::new());
    let target = [edge[0] + 80, edge[1] + 60];
    x11_input::move_pointer(target);
    world.pump(Vec::new());
    x11_input::left_button(target, false);
    world.pump(Vec::new());
    let after = x11_input::application_bounds();
    assert_eq!(
        &after[..2],
        &before[..2],
        "the opposite corner must stay fixed"
    );
    assert!(
        after[2] > before[2] && after[3] > before[3],
        "expected growth: {before:?} -> {after:?}"
    );
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_drag_moves_window_without_resizing() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    let before = x11_input::application_bounds();
    let point = [1050, 18];
    x11_input::move_pointer(point);
    world.pump(Vec::new());
    x11_input::left_button(point, true);
    world.pump(Vec::new());
    x11_input::move_pointer([1070, 30]);
    world.pump(Vec::new());
    x11_input::move_pointer([1120, 80]);
    world.pump(Vec::new());
    x11_input::left_button([1120, 80], false);
    world.pump(Vec::new());
    let after = x11_input::application_bounds();
    assert_ne!(&after[..2], &before[..2], "expected the frame to move");
    assert_eq!(&after[2..], &before[2..], "moving must preserve size");
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_close_button_obeys_the_unsaved_document_guard() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    world.start_sketch();
    x11_input::focus_window();
    world.pump(Vec::new());
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    let close = [width - 24, 17];
    x11_input::move_pointer(close);
    world.pump(Vec::new());
    x11_input::left_button(close, true);
    world.pump(Vec::new());
    x11_input::left_button(close, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        world
            .app()
            .window
            .gui_actions
            .contains(&oblikovati_gui::GuiShellAction::Exit),
        "Close control did not emit Exit"
    );
    world.expect_text("Save changes to");
    let cancel = super::diagnostics::pointer_pixels(world.run("Cancel").center());
    x11_input::click(cancel);
    world.pump(Vec::new());
    assert!(world.document_is_open(document));
    assert!(world.state().document_storage[&document].unsaved_edits);
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_respects_minimum_size() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    let before = x11_input::application_bounds();
    let edge = [before[2] - 2, before[3] - 2];
    x11_input::move_pointer(edge);
    world.pump(Vec::new());
    x11_input::left_button(edge, true);
    world.pump(Vec::new());
    x11_input::move_pointer([100, 100]);
    world.pump(Vec::new());
    x11_input::left_button([100, 100], false);
    world.pump(Vec::new());
    let after = x11_input::application_bounds();
    assert_eq!(&after[2..], &[640, 480]);
    press_maximize(&mut world);
    pump_until(&mut world, true);
    world.close();
}

fn drag_resize_handle(world: &mut ApplicationWorld, point: [i32; 2], target: [i32; 2]) {
    x11_input::move_pointer(point);
    world.pump(Vec::new());
    x11_input::left_button(point, true);
    world.pump(Vec::new());
    x11_input::move_pointer(target);
    world.pump(Vec::new());
    x11_input::left_button(target, false);
    world.pump(Vec::new());
}
fn handle_coordinate(direction: i32, extent: i32) -> i32 {
    match direction {
        -1 => 1,
        1 => extent - 2,
        _ => extent / 2,
    }
}
fn verify_resize_axis(before: [i32; 4], after: [i32; 4], axis: usize, direction: i32) {
    assert_eq!(
        after[axis + 2] > before[axis + 2],
        direction != 0,
        "resize axis {axis}, direction {direction}: {before:?} -> {after:?}"
    );
    if direction < 0 {
        assert_eq!(
            before[axis] + before[axis + 2],
            after[axis] + after[axis + 2]
        );
    }
    if direction > 0 {
        assert_eq!(before[axis], after[axis]);
    }
}
#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_all_eight_resize_handles_grow_their_axes() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    for (horizontal, vertical) in [
        (-1, 0),
        (1, 0),
        (0, -1),
        (0, 1),
        (-1, -1),
        (1, 1),
        (1, -1),
        (-1, 1),
    ] {
        let before = x11_input::application_bounds();
        let point = [
            handle_coordinate(horizontal, before[2]),
            handle_coordinate(vertical, before[3]),
        ];
        let target = [point[0] + horizontal * 20, point[1] + vertical * 20];
        drag_resize_handle(&mut world, point, target);
        let after = x11_input::application_bounds();
        verify_resize_axis(before, after, 0, horizontal);
        verify_resize_axis(before, after, 1, vertical);
    }
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn native_frame_double_click_toggles_maximize() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    for maximized in [true, false] {
        x11_input::click([1050, 18]);
        x11_input::click([1050, 18]);
        world.pump(Vec::new());
        pump_until(&mut world, maximized);
    }
    world.close();
}
