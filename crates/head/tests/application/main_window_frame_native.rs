// SPDX-License-Identifier: GPL-2.0-only
//! Window-manager behavior for the application-owned frame (#4063).
use super::{ApplicationWorld, x11_input};
use std::time::{Duration, Instant};

pub(super) fn pump_until(world: &mut ApplicationWorld, expected: bool) {
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
pub(super) fn press_maximize(world: &mut ApplicationWorld) {
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

pub(super) fn drag_resize_handle(world: &mut ApplicationWorld, point: [i32; 2], target: [i32; 2]) {
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
pub(super) fn verify_resize_axis(before: [i32; 4], after: [i32; 4], axis: usize, direction: i32) {
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

#[test]
#[ignore = "requires an X11 window manager"]
fn desktop_close_shortcut_obeys_the_unsaved_document_guard() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    world.start_sketch();
    x11_input::press_alt_f4();
    world.pump(Vec::new());
    world.pump(Vec::new());
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
fn maximized_frame_ignores_resize_edges() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    press_maximize(&mut world);
    pump_until(&mut world, true);
    let before = x11_input::application_bounds();
    drag_resize_handle(&mut world, [1, before[3] / 2], [40, before[3] / 2]);
    assert_eq!(x11_input::application_bounds(), before);
    assert!(world.app().window.native.view.window_maximized().unwrap());
    world.close();
}

pub(super) fn wait_for_hidden(hidden: bool) {
    let deadline = Instant::now() + Duration::from_secs(5);
    while x11_input::window_has_state(b"_NET_WM_STATE_HIDDEN") != hidden {
        assert!(Instant::now() < deadline, "expected hidden={hidden}");
        std::thread::sleep(Duration::from_millis(30));
    }
}
pub(super) fn pump_minimize_button(world: &mut ApplicationWorld) {
    use crate::HeadWindow;
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    x11_input::click([width - 120, 17]);
    let deadline = Instant::now() + Duration::from_secs(5);
    while !x11_input::window_has_state(b"_NET_WM_STATE_HIDDEN") {
        let app = world.app();
        let events = app.window.events(64).unwrap();
        for event in events {
            crate::frames::advance_gui_for_native_test(
                &mut app.window,
                &mut app.owner,
                Some(&event),
            )
            .unwrap();
        }
        crate::frames::advance_gui_for_native_test(&mut app.window, &mut app.owner, None).unwrap();
        assert!(
            Instant::now() < deadline,
            "Minimize control: expected the window to become hidden"
        );
        std::thread::sleep(Duration::from_millis(30));
    }
}
fn exercise_minimize_reactivation(use_taskbar: bool) {
    for maximized in [false, true] {
        let mut world = ApplicationWorld::new();
        let document = world.add_part();
        let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
        x11_input::focus_window();
        if maximized {
            press_maximize(&mut world);
            pump_until(&mut world, true);
        }
        pump_minimize_button(&mut world);
        assert!(!x11_input::window_has_state(b"_NET_WM_STATE_SKIP_TASKBAR"));
        if use_taskbar {
            std::thread::sleep(Duration::from_millis(500));
            x11_input::click_test_taskbar();
        } else {
            x11_input::activate_from_taskbar();
        }
        wait_for_hidden(false);
        world.pump(Vec::new());
        pump_until(&mut world, maximized);
        assert_eq!(world.active_id(), document);
        assert_eq!(
            crate::HeadWindow::view_camera(&world.app().window).unwrap(),
            camera
        );
        world.close();
    }
}
#[test]
#[ignore = "requires an X11 window manager"]
fn minimize_button_keeps_taskbar_entry_and_restores_window() {
    exercise_minimize_reactivation(false);
}
#[test]
#[ignore = "requires Openbox and the dedicated tint2 test panel"]
fn minimize_button_restores_by_clicking_the_taskbar() {
    exercise_minimize_reactivation(true);
}

#[test]
#[ignore = "requires an X11 window manager and XFixes"]
fn resize_handles_show_direction_specific_cursors() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    let bounds = x11_input::application_bounds();
    let handles = [
        ([1, bounds[3] / 2], 70),
        ([bounds[2] - 2, bounds[3] / 2], 96),
        ([bounds[2] / 2, 1], 138),
        ([bounds[2] / 2, bounds[3] - 2], 16),
        ([1, 1], 134),
        ([bounds[2] - 2, bounds[3] - 2], 14),
        ([bounds[2] - 2, 1], 136),
        ([1, bounds[3] - 2], 12),
    ];
    for (point, shape) in handles {
        let expected = x11_input::reference_cursor(shape);
        x11_input::move_pointer(point);
        world.pump(Vec::new());
        let actual = x11_input::cursor_sprite();
        assert_eq!(
            actual, expected,
            "cursor for {point:?}: expected X cursor shape {shape}"
        );
    }
    world.close();
}
