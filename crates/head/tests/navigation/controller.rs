// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn pointer_button(button: ViewportButton, pressed: bool, position: [i32; 2]) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button,
        pressed,
        position,
        modifiers: Default::default(),
    })
}
fn pointer_motion(position: [i32; 2], shift: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers: ViewportModifiers {
            shift,
            control: true,
        },
    })
}
fn held_key(key: ViewportKey, pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Key { key, pressed })
}
fn command_after_drag(
    button: ViewportButton,
    modal: Option<ViewportKey>,
    shift: bool,
) -> Option<ViewportNavigation> {
    let mut navigation = DesktopNavigation::default();
    if let Some(key) = modal {
        navigation.command(&held_key(key, true)).unwrap();
    }
    navigation
        .command(&pointer_button(button, true, [10, 20]))
        .unwrap();
    navigation
        .command(&pointer_motion([20, 30], shift))
        .unwrap()
}
fn start_middle_drag() -> DesktopNavigation {
    let mut navigation = DesktopNavigation::default();
    navigation
        .command(&pointer_button(ViewportButton::Middle, true, [10, 20]))
        .unwrap();
    navigation
}
fn advance_middle_drag(
    navigation: &mut DesktopNavigation,
    position: [i32; 2],
) -> Option<ViewportNavigation> {
    navigation
        .command(&pointer_motion(position, false))
        .unwrap()
}
#[test]
fn native_defaults_keep_plain_left_and_right_drags_out_of_navigation() {
    assert_eq!(command_after_drag(ViewportButton::Left, None, false), None);
    assert_eq!(command_after_drag(ViewportButton::Right, None, false), None);
    assert_eq!(
        command_after_drag(ViewportButton::Middle, None, false),
        Some(ViewportNavigation::Pan([10, 10]))
    );
    assert_eq!(
        command_after_drag(ViewportButton::Middle, None, true),
        Some(ViewportNavigation::Orbit {
            yaw: -0.1,
            pitch: -0.1
        })
    );
}
#[test]
fn held_function_keys_route_left_drags_to_the_go_navigation_modes() {
    assert_eq!(
        command_after_drag(ViewportButton::Left, Some(ViewportKey::Pan), false),
        Some(ViewportNavigation::Pan([10, 10]))
    );
    assert_eq!(
        command_after_drag(ViewportButton::Left, Some(ViewportKey::Orbit), false),
        Some(ViewportNavigation::Orbit {
            yaw: -0.1,
            pitch: -0.1
        })
    );
    assert_eq!(
        command_after_drag(ViewportButton::Left, Some(ViewportKey::Zoom), false),
        Some(ViewportNavigation::Zoom {
            factor: 1.01_f64.powi(10),
            anchor: None
        })
    );
}
#[test]
fn wheel_input_preserves_the_cursor_anchor_and_go_notch_factor() {
    let mut navigation = DesktopNavigation::default();
    let event = ViewportEvent::Input(ViewportInput::Wheel {
        notches: 2,
        position: [12, 24],
    });
    assert_eq!(
        navigation.command(&event).unwrap(),
        Some(ViewportNavigation::Zoom {
            factor: 0.9_f64.powi(2),
            anchor: Some([12, 24])
        })
    );
}
#[test]
fn matching_release_focus_unmap_close_and_escape_end_a_drag() {
    for stop in [
        pointer_button(ViewportButton::Middle, false, [10, 20]),
        ViewportEvent::Focus(false),
        ViewportEvent::Mapped(false),
        ViewportEvent::CloseRequested,
        held_key(ViewportKey::Cancel, true),
    ] {
        let mut navigation = start_middle_drag();
        navigation.command(&stop).unwrap();
        assert_eq!(
            navigation
                .command(&pointer_motion([30, 40], false))
                .unwrap(),
            None
        );
    }
}
#[test]
fn unrelated_releases_and_stationary_motion_do_not_steal_a_captured_drag() {
    let mut navigation = start_middle_drag();
    navigation
        .command(&pointer_button(ViewportButton::Left, false, [10, 20]))
        .unwrap();
    navigation
        .command(&held_key(ViewportKey::Cancel, false))
        .unwrap();
    assert_eq!(
        advance_middle_drag(&mut navigation, [10, 20]),
        Some(ViewportNavigation::Pan([0, 0]))
    );
    assert_eq!(
        advance_middle_drag(&mut navigation, [-10, 90]),
        Some(ViewportNavigation::Pan([-20, 70]))
    );
    assert_eq!(navigation.command(&ViewportEvent::Expose).unwrap(), None);
}
#[test]
fn key_release_and_multiple_held_modes_preserve_the_modal_priority() {
    let mut navigation = DesktopNavigation::default();
    for key in [ViewportKey::Orbit, ViewportKey::Zoom, ViewportKey::Pan] {
        navigation.command(&held_key(key, true)).unwrap();
    }
    navigation
        .command(&pointer_button(ViewportButton::Left, true, [0, 0]))
        .unwrap();
    assert_eq!(
        navigation.command(&pointer_motion([1, 1], false)).unwrap(),
        Some(ViewportNavigation::Pan([1, 1]))
    );
    for key in [ViewportKey::Pan, ViewportKey::Zoom, ViewportKey::Orbit] {
        navigation.command(&held_key(key, false)).unwrap();
    }
    assert_eq!(
        navigation.command(&pointer_motion([2, 2], false)).unwrap(),
        None
    );
}
#[test]
fn pointer_overflow_is_refused_without_changing_the_drag_anchor() {
    let mut navigation = DesktopNavigation::default();
    navigation
        .command(&pointer_button(ViewportButton::Middle, true, [i32::MIN, 0]))
        .unwrap();
    let error = navigation
        .command(&pointer_motion([i32::MAX, 0], false))
        .unwrap_err();
    assert!(error.contains("expected representable signed 32-bit pixel deltas"));
    assert_eq!(
        navigation
            .command(&pointer_motion([i32::MIN + 1, 0], false))
            .unwrap(),
        Some(ViewportNavigation::Pan([1, 0]))
    );
}

#[test]
fn home_press_clears_the_drag_and_release_does_not_repeat_the_command() {
    let mut navigation = start_middle_drag();
    assert_eq!(
        navigation
            .command(&held_key(ViewportKey::Home, true))
            .unwrap(),
        Some(ViewportNavigation::Home { margin: 0.1 })
    );
    assert_eq!(
        navigation
            .command(&held_key(ViewportKey::Home, false))
            .unwrap(),
        None
    );
    assert_eq!(advance_middle_drag(&mut navigation, [20, 30]), None);
}

#[test]
fn previous_view_cancels_a_drag_without_producing_a_continuous_camera_command() {
    let mut navigation = start_middle_drag();
    assert_eq!(
        navigation
            .command(&held_key(ViewportKey::Previous, true))
            .unwrap(),
        None
    );
    assert_eq!(advance_middle_drag(&mut navigation, [20, 30]), None);
    assert_eq!(
        navigation
            .command(&held_key(ViewportKey::Previous, false))
            .unwrap(),
        None
    );
}
