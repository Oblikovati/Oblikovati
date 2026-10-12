// SPDX-License-Identifier: GPL-2.0-only
use super::scene_tests::{capture_native_scene, open_native_part, publish_native_scene};
use super::*;
use crate::HeadWork;
use oblikovati_viewport::{ViewportButton, ViewportInput, ViewportModifiers};
fn drag_input(button: ViewportButton, position: [i32; 2]) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button,
        pressed: true,
        position,
        modifiers: Default::default(),
    })
}
fn motion_input(position: [i32; 2], shift: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers: ViewportModifiers {
            shift,
            control: false,
        },
    })
}
fn apply_owner_drag(window: &mut NativeHeadWindow, button: ViewportButton) -> bool {
    let inputs = [
        drag_input(button, [100, 100]),
        motion_input([120, 130], false),
    ];
    window.apply_navigation_inputs(&inputs).unwrap()
}
#[test]
fn native_owner_navigation_changes_only_the_camera_and_clears_on_focus_loss() {
    let mut window = NativeHeadWindow::open().unwrap();
    let initial = window.view.camera_state().unwrap();
    assert!(!apply_owner_drag(&mut window, ViewportButton::Left));
    assert_eq!(window.view.camera_state().unwrap(), initial);
    assert!(apply_owner_drag(&mut window, ViewportButton::Middle));
    let panned = window.view.camera_state().unwrap();
    assert_ne!(panned, initial);
    let cancelled = [ViewportEvent::Focus(false), motion_input([140, 160], true)];
    assert!(!window.apply_navigation_inputs(&cancelled).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), panned);
    window.finish().unwrap();
}
#[test]
fn native_owner_dispatch_propagates_invalid_camera_input() {
    let mut window = NativeHeadWindow::open().unwrap();
    let invalid = ViewportEvent::Input(ViewportInput::Wheel {
        notches: i32::MAX,
        position: [100, 100],
    });
    let initial = window.view.camera_state().unwrap();
    assert!(window.apply_navigation_inputs(&[invalid]).is_err());
    assert_eq!(window.view.camera_state().unwrap(), initial);
    window.view.close();
    let wheel = ViewportEvent::Input(ViewportInput::Wheel {
        notches: 1,
        position: [100, 100],
    });
    assert!(window.apply_navigation_inputs(&[wheel]).is_err());
    window.finish().unwrap();
}

#[test]
fn navigation_renders_the_current_part_without_new_model_evaluation() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("navigation.opd"));
    let before = publish_native_scene(&mut owner, &mut window);
    let inputs = vec![
        drag_input(ViewportButton::Middle, [100, 100]),
        motion_input([120, 130], false),
    ];
    let inputs = window.navigation_events(inputs).unwrap();
    assert_eq!(inputs.last(), Some(&ViewportEvent::Expose));
    let after = capture_native_scene(&mut window);
    assert_ne!(before.rgba, after.rgba);
    assert_eq!(before.scene_generation, after.scene_generation);
    assert!(after.camera_generation > before.camera_generation);
    assert!(owner.take_scene().unwrap().is_none());
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn idle_pointer_inputs_do_not_request_a_frame() {
    let mut window = NativeHeadWindow::open().unwrap();
    let inputs = vec![motion_input([100, 100], false)];
    assert_eq!(window.navigation_events(inputs.clone()).unwrap(), inputs);
    window.finish().unwrap();
}

#[test]
fn native_home_key_recovers_the_part_without_recompute() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("home.opd"));
    let before = publish_native_scene(&mut owner, &mut window);
    recover_native_home(&mut window);
    let restored = capture_native_scene(&mut window);
    assert!(restored.rgba.as_chunks::<4>().0.contains(&[0, 255, 0, 255]));
    assert_eq!(restored.scene_generation, before.scene_generation);
    assert!(owner.take_scene().unwrap().is_none());
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn recover_native_home(window: &mut NativeHeadWindow) {
    window
        .view
        .navigate(oblikovati_viewport::ViewportNavigation::Pan([4000, 4000]))
        .unwrap();
    let home = ViewportEvent::Input(ViewportInput::Key {
        key: oblikovati_viewport::ViewportKey::Home,
        pressed: true,
    });
    assert_eq!(
        window.navigation_events(vec![home]).unwrap().last(),
        Some(&ViewportEvent::Expose)
    );
}
