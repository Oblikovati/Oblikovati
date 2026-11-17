// SPDX-License-Identifier: GPL-2.0-only
//! Part canvas menu uses the user's configured base marking menu (#4045).
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;

const POPUP: &str = "##part-canvas-menu";
pub(crate) fn canvas_menu(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    if state.authoring != GuiAuthoringContext::Part || state.document.is_none() {
        return false;
    }
    if !ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        && ui.is_mouse_released(imgui::MouseButton::Right)
    {
        ui.open_popup(POPUP);
    }
    let Some(_popup) = ui.begin_popup(POPUP) else {
        return false;
    };
    draw_configured_menu(ui, state, actions);
    draw_selected_face_measure(ui, state, actions);
    let look_at = GuiShellAction::LookAtSelection;
    if ui
        .menu_item_config("Look At")
        .enabled(look_at.enabled(state))
        .build()
    {
        actions.push(look_at);
    }
    if ui.menu_item("Home View") {
        actions.push(GuiShellAction::Home);
    }
    true
}
fn draw_selected_face_measure(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(GuiShellAction::StartSketchOnSelectedFace { document, key }) =
        selected_face_action(state)
    else {
        return;
    };
    let action = GuiShellAction::MeasureSelectedFace { document, key };
    if ui
        .menu_item_config("Measure")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}
fn draw_configured_menu(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(menus) = &state.marking_menus else {
        return;
    };
    let menu = menus.get(oblikovati_api::types::BASE_ENVIRONMENT);
    let selected = selected_face_action(state);
    let resolve = if selected.is_some() {
        selected_face_command
    } else {
        command
    };
    let start = actions.len();
    if state.sketch_classic_menu {
        crate::marking_menu::draw_classic(ui, &menu, state, actions, resolve);
    } else {
        crate::marking_menu::draw(ui, &menu, state, actions, resolve);
    }
    for action in &mut actions[start..] {
        if *action == GuiShellAction::ChooseSketchPlane
            && let Some(selected) = &selected
        {
            *action = selected.clone();
        }
    }
}
fn selected_face_action(state: &GuiCommandState) -> Option<GuiShellAction> {
    let document = state.document?;
    let [(key, _)] = state.selected_face_appearances.as_slice() else {
        return None;
    };
    let action = GuiShellAction::StartSketchOnSelectedFace {
        document,
        key: key.clone(),
    };
    action.enabled(state).then_some(action)
}
fn selected_face_command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    if id == "Sketch.Create2D" {
        return Some(("New Sketch", GuiShellAction::ChooseSketchPlane));
    }
    command(id)
}
fn command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    match id {
        "Create.Extrude" => Some(("Extrude", GuiShellAction::OpenExtrude)),
        "Modify.Fillet" => Some(("Fillet", GuiShellAction::OpenFillet)),
        "Modify.Chamfer" => Some(("Chamfer", GuiShellAction::OpenChamfer)),
        "Sketch.Create2D" => Some(("Start 2D Sketch", GuiShellAction::ChooseSketchPlane)),
        "WorkPlane.Offset" => Some(("Offset Plane", GuiShellAction::OpenOffsetPlane)),
        _ => None,
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn one_current_face_produces_a_named_sketch_action() {
        let mut state = GuiCommandState {
            document: Some(12),
            authoring: GuiAuthoringContext::Part,
            model_face_chosen: true,
            selected_face_appearances: vec![("face-1".into(), false)],
            ..Default::default()
        };
        assert_eq!(
            selected_face_action(&state),
            Some(GuiShellAction::StartSketchOnSelectedFace {
                document: 12,
                key: "face-1".into()
            })
        );
        let original = selected_face_action(&state).unwrap();
        let measure = GuiShellAction::MeasureSelectedFace {
            document: 12,
            key: "face-1".into(),
        };
        assert!(measure.enabled(&state));
        state.selected_face_appearances[0].0 = "face-2".into();
        assert!(!measure.enabled(&state));
        assert!(
            !original.enabled(&state),
            "a changed face must not execute a stale action"
        );
        state.selected_face_appearances[0].0 = "face-1".into();
        state.document = Some(13);
        assert!(
            !original.enabled(&state),
            "a different document must not execute a stale action"
        );
        state.document = Some(12);
        assert_eq!(
            selected_face_command("Sketch.Create2D"),
            Some(("New Sketch", GuiShellAction::ChooseSketchPlane))
        );
        state
            .selected_face_appearances
            .push(("face-2".into(), false));
        assert_eq!(selected_face_action(&state), None);
        state.selected_face_appearances.pop();
        state.selected_body_keys.push("body".into());
        assert_eq!(selected_face_action(&state), None);
        state.selected_body_keys.clear();
        state.selected_edge_count = 1;
        assert_eq!(selected_face_action(&state), None);
        state.selected_edge_count = 0;
        state.transaction_open = true;
        assert_eq!(selected_face_action(&state), None);
    }
    #[test]
    fn part_commands_resolve_only_known_actions() {
        assert_eq!(
            command("Modify.Fillet"),
            Some(("Fillet", GuiShellAction::OpenFillet))
        );
        assert_eq!(
            command("Create.Extrude"),
            Some(("Extrude", GuiShellAction::OpenExtrude))
        );
        assert_eq!(
            command("Modify.Chamfer"),
            Some(("Chamfer", GuiShellAction::OpenChamfer))
        );
        assert_eq!(command("Modify.Unknown"), None);
    }
}
