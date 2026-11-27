// SPDX-License-Identifier: GPL-2.0-only
use super::*;

/// The camera's Home, as the View tab's Navigate panel sends it; the row's Home opens a workspace.
pub(super) fn press_camera_home(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Key {
        key: oblikovati_viewport::ViewportKey::Home,
        pressed: true,
    })];
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
}
pub(super) fn verify_gui_camera(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window
        .native
        .navigate_view(oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 })
        .unwrap();
    let initial = window.native.view.camera_state().unwrap();
    let initial_pixel = window.native.view.model_pixel([0., 0., 0.]).unwrap();
    assert!(!drive_wheel(window, owner, [50, 50]));
    assert_eq!(window.native.view.camera_state().unwrap(), initial);
    assert!(drive_wheel(window, owner, [600, 400]));
    assert_ne!(window.native.view.camera_state().unwrap(), initial);
    press_camera_home(window, owner);
    assert_eq!(window.native.view.camera_state().unwrap(), initial);
    assert_eq!(
        window.native.view.model_pixel([0., 0., 0.]).unwrap(),
        initial_pixel
    );
}
pub(super) fn drive_wheel(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    position: [i32; 2],
) -> bool {
    let wheel = ViewportEvent::Input(ViewportInput::Wheel {
        notches: 2,
        position,
    });
    window.scripted = vec![wheel];
    window.passed.clear();
    crate::frames::run_frames(window, HeadOptions { frames: Some(2) }, owner).unwrap();
    window.passed.contains(&wheel)
}

pub(super) fn assert_model_point_in_document(window: &mut ScriptedGuiWindow) {
    let extent = window.native.view.drawable_extent().unwrap();
    let region = window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    let [x, y] = window.native.view.model_pixel([0., 0., 0.]).unwrap();
    assert!(
        x >= region[0] as i32
            && x < region[2] as i32
            && y >= region[1] as i32
            && y < region[3] as i32
    );
}
