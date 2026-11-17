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
    if let Some(menus) = &state.marking_menus {
        let menu = menus.get(oblikovati_api::types::BASE_ENVIRONMENT);
        if state.sketch_classic_menu {
            crate::marking_menu::draw_classic(ui, &menu, state, actions, command);
        } else {
            crate::marking_menu::draw(ui, &menu, state, actions, command);
        }
    }
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
fn command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    match id {
        "Create.Extrude" => Some(("Extrude", GuiShellAction::OpenExtrude)),
        "Modify.Fillet" => Some(("Fillet", GuiShellAction::OpenFillet)),
        "Sketch.Create2D" => Some(("Start 2D Sketch", GuiShellAction::ChooseSketchPlane)),
        "WorkPlane.Offset" => Some(("Offset Plane", GuiShellAction::OpenOffsetPlane)),
        _ => None,
    }
}
#[cfg(test)]
mod tests {
    use super::*;
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
        assert_eq!(command("Modify.Unknown"), None);
    }
}
