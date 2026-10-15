// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadWindow, HeadWork, placement_test_inputs::*};
use oblikovati_gui::GuiShellAction;

#[test]
fn native_cube_actions_project_mouse_input_and_render_a_solid() {
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    for action in [
        GuiShellAction::NewPart,
        GuiShellAction::SketchXY,
        GuiShellAction::Rectangle,
    ] {
        assert!(owner.apply_gui_shell_action(action).unwrap());
    }
    window.set_sketch_edit(owner.active_sketch_edit()).unwrap();
    owner
        .process_window_inputs(
            &[button([650, 430], true), button([650, 430], false)],
            &window,
        )
        .unwrap();
    owner
        .process_window_inputs(&[motion([820, 520]), button([820, 520], true)], &window)
        .unwrap();
    assert!(!owner.placement_active());
    assert!(
        owner
            .apply_gui_shell_action(GuiShellAction::Extrude)
            .unwrap()
    );
    window.set_sketch_edit(None).unwrap();
    let capture = super::publish_native_scene(&mut owner, &mut window);
    assert!(super::solid_pixels(&capture) > 0);
    owner.finish().unwrap();
    window.finish().unwrap();
}
