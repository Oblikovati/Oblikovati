// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[test]
fn save_notice_before_initial_gui_frame_keeps_the_requesting_document() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("startup.opd");
    let (mut window, mut owner) = initial_save_window();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    type_native_text(&mut window, &mut owner, path.to_str().unwrap());
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Enter);
    assert!(path.is_file());
    assert!(owner.close_allowed().unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn initial_save_window() -> (ScriptedGuiWindow, NativeDesktopOwner) {
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    native
        .show_command_notice(crate::DesktopCommandNotice::SavePathRequired { document })
        .unwrap();
    (scripted_window(native), owner)
}

pub(super) fn save_rectangle_file(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) -> tempfile::TempDir {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("rectangle.opd");
    cancel_save_dialog(window, owner);
    save_from_dialog(window, owner, &path);
    parameters::verify_rectangle(owner, edit, 2.);
    assert!(path.is_file());
    assert_eq!(owner.documents().path(edit.document).unwrap(), path);
    assert!(owner.close_allowed().unwrap());
    close_and_reopen(window, owner, edit, &path);
    directory
}
fn choose_file_menu(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner, item_y: i32) {
    click_shell_position(window, owner, [195, 107]);
    click_shell_position(window, owner, [340, item_y]);
}
fn close_and_reopen(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
    path: &std::path::Path,
) {
    choose_file_menu(window, owner, 173);
    assert!(owner.documents().workspace().active_document().is_none());
    choose_file_menu(window, owner, 122);
    type_native_text(window, owner, path.to_str().unwrap());
    press_shell_key(window, owner, ViewportPhysicalKey::Enter);
    let document = owner.documents().workspace().active_document().unwrap();
    assert_ne!(document, edit.document);
    parameters::verify_rectangle(owner, crate::HeadSketchEdit { document, ..edit }, 2.);
    assert!(owner.close_allowed().unwrap());
}
fn cancel_save_dialog(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let before = owner.gui_command_state().unwrap();
    click_shell(window, owner, 30);
    press_shell_key(window, owner, ViewportPhysicalKey::Enter);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    press_shell_key(window, owner, ViewportPhysicalKey::Escape);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert!(!owner.close_allowed().unwrap());
}
fn save_from_dialog(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    path: &std::path::Path,
) {
    click_shell(window, owner, 30);
    type_native_text(window, owner, path.to_str().unwrap());
    press_shell_key(window, owner, ViewportPhysicalKey::Enter);
    assert!(matches!(
        window.native.command_notice,
        Some(crate::DesktopCommandNotice::Saved { .. })
    ));
}
