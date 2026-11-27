// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn click_shell_command(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    label: &str,
    expected: oblikovati_gui::GuiShellAction,
) {
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let position = application_bdd::rendered_label_center(label);
    window.gui_actions.clear();
    click_shell_position(window, owner, position);
    assert!(
        window.gui_actions.contains(&expected),
        "{label}: {:?}",
        window.gui_actions
    );
}
#[test]
fn native_gui_buttons_save_and_restore_model_pixels_without_camera_leakage() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("gui commands.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    let before = publish_gui_framebuffer(&mut owner, &mut window.native);
    native_set(&mut owner, "width", "6 cm");
    let edited = publish_gui_framebuffer(&mut owner, &mut window.native);
    assert_ne!(before.rgba, edited.rgba);
    click_shell_command(
        &mut window,
        &mut owner,
        "Undo",
        oblikovati_gui::GuiShellAction::Undo,
    );
    assert_model_without_gui(&mut window.native, &before);
    click_shell_command(
        &mut window,
        &mut owner,
        "Redo",
        oblikovati_gui::GuiShellAction::Redo,
    );
    assert_model_without_gui(&mut window.native, &edited);
    click_shell_command(
        &mut window,
        &mut owner,
        "Save",
        oblikovati_gui::GuiShellAction::Save,
    );
    verify_gui_save(&path, &mut window.native);
    verify_gui_camera(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn physical_keyboard_navigation_activates_native_save_without_camera_input() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("keyboard commands.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    // Saving leaves keyboard focus on the row's Save button; the edit then has something to save.
    click_shell_command(
        &mut window,
        &mut owner,
        "Save",
        oblikovati_gui::GuiShellAction::Save,
    );
    native_set(&mut owner, "width", "6 cm");
    verify_keyboard_save_navigation(&mut window, &mut owner, &path);
    verify_viewport_arrow_navigation(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_keyboard_save_navigation(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    path: &std::path::Path,
) {
    let camera = window.native.view.camera_state().unwrap();
    for key in [
        oblikovati_viewport::ViewportPhysicalKey::RightArrow,
        oblikovati_viewport::ViewportPhysicalKey::LeftArrow,
    ] {
        window.passed.clear();
        press_shell_key(window, owner, key);
        assert!(
            !window.passed.iter().any(|event| matches!(
                event,
                ViewportEvent::Input(ViewportInput::PhysicalKey { pressed: true, .. })
            )),
            "toolbar navigation {key:?} reached the viewport"
        );
        assert_eq!(window.native.view.camera_state().unwrap(), camera);
    }
    press_shell_key(
        window,
        owner,
        oblikovati_viewport::ViewportPhysicalKey::Space,
    );
    verify_gui_save(path, &mut window.native);
    assert_eq!(window.native.view.camera_state().unwrap(), camera);
}
fn verify_viewport_arrow_navigation(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) {
    window.scripted = pointer_click_events([1100, 800], 1);
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let before = window.native.view.camera_state().unwrap();
    press_shell_key(
        window,
        owner,
        oblikovati_viewport::ViewportPhysicalKey::RightArrow,
    );
    assert_ne!(window.native.view.camera_state().unwrap(), before);
}
fn assert_model_without_gui(
    window: &mut NativeHeadWindow,
    expected: &oblikovati_viewport::ViewportCapture,
) {
    let composed = window.view.capture().unwrap();
    assert_ne!(composed.rgba, expected.rgba);
    assert!(window.command_notice.is_some());
    window.clear_gui_frame().unwrap();
    assert!(window.view.capture().is_err());
    scene_tests::assert_same_native_pixels(&capture_framebuffer(window), expected);
}
fn verify_gui_save(path: &std::path::Path, window: &mut NativeHeadWindow) {
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(path).unwrap())
            .unwrap();
    let restored = package.restore_document_parameters().unwrap();
    let width = restored.parameters.find("width").unwrap();
    assert_eq!(restored.parameters.get(width).unwrap().source, "6 cm");
    assert!(window.view.native_title().unwrap().contains("Saved"));
}
