// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::{HeadWork, placement_test_inputs::*};
use oblikovati_gui::GuiShellAction;

#[test]
fn gui_document_actions_select_the_document_specific_ribbon() {
    let (mut owner, _) = owner();
    for (action, key, tab) in [
        (GuiShellAction::NewPart, "Part", "Create & Modify"),
        (GuiShellAction::NewAssembly, "Assembly", "Assemble"),
        (GuiShellAction::NewDrawing, "Drawing", "Drawing"),
    ] {
        assert!(owner.apply_gui_shell_action(action).unwrap());
        let state = owner.gui_command_state().unwrap();
        assert_eq!(state.ribbon_key, key);
        assert_eq!(state.ribbon_tabs[0].name, tab);
    }
}

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
            .apply_gui_shell_action(GuiShellAction::ExitSketch)
            .unwrap()
    );
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

#[test]
fn model_browser_visibility_action_updates_the_current_body() {
    let (mut owner, _) = owner();
    for action in [
        GuiShellAction::NewPart,
        GuiShellAction::SketchXY,
        GuiShellAction::Rectangle,
    ] {
        owner.apply_gui_shell_action(action).unwrap();
    }
    deliver(&mut owner, &[button([0, 0], true), button([0, 0], false)]);
    deliver(&mut owner, &[motion([10, 10]), button([10, 10], true)]);
    owner
        .apply_gui_shell_action(GuiShellAction::Extrude)
        .unwrap();
    let body = owner
        .gui_command_state()
        .unwrap()
        .model_tree
        .unwrap()
        .bodies[0]
        .clone();
    assert!(
        owner
            .apply_gui_shell_action(GuiShellAction::SetBodyVisible {
                index: body.index,
                visible: false,
            })
            .unwrap()
    );
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .model_tree
            .unwrap()
            .bodies[0]
            .visible
    );
}
