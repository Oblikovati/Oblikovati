// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::{HeadWork, placement_test_inputs::*};
use oblikovati_gui::GuiShellAction;

#[test]
fn gui_cube_actions_create_a_mouse_drawn_cube_in_the_part_viewport() {
    let (mut owner, _) = owner();
    for action in [
        GuiShellAction::NewPart,
        GuiShellAction::SketchXY,
        GuiShellAction::Rectangle,
    ] {
        assert!(owner.apply_gui_shell_action(action).unwrap());
    }
    deliver(&mut owner, &[button([0, 0], true), button([0, 0], false)]);
    deliver(&mut owner, &[motion([10, 10]), button([10, 10], true)]);
    assert!(!owner.placement_active());
    assert!(
        owner
            .apply_gui_shell_action(GuiShellAction::Extrude)
            .unwrap()
    );
    let update = [owner.take_scene().unwrap(), owner.take_scene().unwrap()]
        .into_iter()
        .flatten()
        .max_by_key(|update| update.status.visible_bodies)
        .unwrap();
    assert_eq!(update.status.visible_bodies, 1);
    assert!(!update.scene.triangles.is_empty());
}
