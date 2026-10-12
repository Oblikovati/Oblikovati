// SPDX-License-Identifier: GPL-2.0-only
use super::scene_tests::{
    capture_native_scene, native_set, open_native_part, publish_native_scene,
};
use super::*;
use crate::HeadWork;
use oblikovati_viewport::{ViewportCapture, ViewportInput, ViewportKey};
#[test]
fn native_history_keys_restore_prism_pixels_without_changing_the_camera() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_script_owner(window.wake.clone()).unwrap();
    open_native_part(
        owner.document_owner_mut(),
        &directory.path().join("history.opd"),
    );
    let before = publish_native_scene(owner.document_owner_mut(), &mut window);
    native_set(owner.document_owner_mut(), "width", "6 cm");
    let edited = publish_native_scene(owner.document_owner_mut(), &mut window);
    assert_ne!(before.rgba, edited.rgba);
    verify_history_pixels(&mut owner, &mut window, ViewportKey::Undo, &before);
    verify_history_pixels(&mut owner, &mut window, ViewportKey::Redo, &edited);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_history_pixels(
    owner: &mut NativeScriptDesktop,
    window: &mut NativeHeadWindow,
    key: ViewportKey,
    expected: &ViewportCapture,
) {
    let camera = window.view.camera_state().unwrap();
    let input = ViewportEvent::Input(ViewportInput::Key { key, pressed: true });
    assert!(owner.process_inputs(&[input]).unwrap());
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let status_frame = capture_native_scene(window);
    assert!(
        status_frame
            .rgba
            .as_chunks::<4>()
            .0
            .contains(&[255, 255, 255, 255])
    );
    // Validate the overlay first, then retire it to retain the exact whole-model image oracle.
    window.view.set_status_text("").unwrap();
    let capture = capture_native_scene(window);
    assert_eq!(capture.rgba, expected.rgba);
    assert_eq!(window.view.camera_state().unwrap(), camera);
    assert!(window.view.native_title().unwrap().contains("complete"));
}
