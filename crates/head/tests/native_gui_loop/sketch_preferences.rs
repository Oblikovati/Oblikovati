// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn sketch_preferences_change_the_owned_application_options() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for label in ["Tools", "Application Options..."] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    window.gui_actions.clear();
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Show grid in sketch"),
    );
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::SetSketchOptions(_)))
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}
