// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::{HeadWork, placement_test_inputs::*};
use oblikovati_gui::{
    GuiFrameEngine, GuiFrameRequest, GuiInputEvent, GuiPointerButton, GuiShellAction,
};

#[test]
fn gui_document_actions_select_the_document_specific_ribbon() {
    let (mut owner, _) = owner();
    let zero_document = owner.gui_command_state().unwrap();
    assert_eq!(zero_document.ribbon_key, "ZeroDoc");
    assert_eq!(zero_document.ribbon_tabs[0].name, "Get Started");
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
fn ribbon_mouse_journey_creates_a_cube_through_the_host() {
    let (mut owner, _) = owner();
    let mut gui = GuiFrameEngine::new().unwrap();
    for control in ["GetStarted.NewPart", "Sketch.Create", "Sketch.Rectangle"] {
        click_ribbon_control(&mut owner, &mut gui, control);
    }
    deliver(&mut owner, &[button([0, 0], true), button([0, 0], false)]);
    deliver(&mut owner, &[motion([10, 10]), button([10, 10], true)]);
    for control in ["Sketch.Finish", "Feature.Extrude"] {
        click_ribbon_control(&mut owner, &mut gui, control);
    }
    let scene = owner.take_scene().unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 1);
    assert!(!scene.scene.triangles.is_empty());
}

fn click_ribbon_control(owner: &mut TestOwner, gui: &mut GuiFrameEngine, command_id: &str) {
    gui_frame(owner, gui, Vec::new());
    let targets = gui_frame(owner, gui, Vec::new()).targets;
    let target = targets
        .iter()
        .cloned()
        .into_iter()
        .find(|target| target.command_id == command_id && target.enabled)
        .unwrap_or_else(|| {
            let visible = targets
                .iter()
                .map(|target| target.command_id.as_str())
                .collect::<Vec<_>>();
            panic!("ribbon command {command_id:?}: expected an enabled target among {visible:?}")
        });
    let mut actions =
        gui_frame(owner, gui, vec![GuiInputEvent::PointerMoved(target.center)]).actions;
    actions.extend(gui_frame(owner, gui, vec![mouse_button(true)]).actions);
    actions.extend(gui_frame(owner, gui, vec![mouse_button(false)]).actions);
    assert_eq!(actions.len(), 1, "ribbon command {command_id:?}");
    assert!(owner.apply_gui_shell_action(actions.remove(0)).unwrap());
}

fn gui_frame(
    owner: &mut TestOwner,
    gui: &mut GuiFrameEngine,
    events: Vec<GuiInputEvent>,
) -> oblikovati_gui::GuiFrameOutput {
    gui.frame(&GuiFrameRequest {
        extent: [1440, 900],
        events,
        commands: owner.gui_command_state().unwrap(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    })
    .unwrap()
}

fn mouse_button(pressed: bool) -> GuiInputEvent {
    GuiInputEvent::PointerButton {
        button: GuiPointerButton::Left,
        pressed,
    }
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
