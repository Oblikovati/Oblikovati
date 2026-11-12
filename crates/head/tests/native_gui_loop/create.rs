// SPDX-License-Identifier: GPL-2.0-only
//! Create the journey's part with the same chooser a user sees.
use super::*;
use oblikovati_viewport::{ViewportModifiers, ViewportPhysicalKey};

pub(super) fn metric_part(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let key = ViewportPhysicalKey::N;
    let modifiers = ViewportModifiers {
        control: true,
        ..Default::default()
    };
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                command: None,
                pressed,
                modifiers: if pressed {
                    modifiers
                } else {
                    Default::default()
                },
            })
        })
        .into();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    for label in ["Metric", "Standard metric part", "Create"] {
        let position = application_bdd::rendered_label_center(label);
        click_shell_position(window, owner, position);
    }
}

#[test]
fn parameter_editor_accepts_scaled_pointer_input() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    metric_part(&mut window, &mut owner);
    parameters::create_size_parameter(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}
