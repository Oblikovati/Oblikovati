// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction, GuiSketchTool, widget_abi};
use imgui::Ui;

pub(crate) fn circular_tool_button(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    primary: GuiSketchTool,
) {
    let alternate = match primary {
        GuiSketchTool::Circle => Some(GuiSketchTool::ThreePointCircle),
        GuiSketchTool::CenterPointArc => Some(GuiSketchTool::ThreePointArc),
        _ => None,
    };
    let size = if alternate.is_some() {
        [90., 18.]
    } else {
        [108., 18.]
    };
    super::chrome::sketch_tool_button(ui, state, actions, primary, size);
    if let Some(alternate) = alternate {
        circular_tool_menu(ui, state, actions, primary, alternate);
    }
}
fn circular_tool_menu(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    primary: GuiSketchTool,
    alternate: GuiSketchTool,
) {
    let identity = format!("##{:?}-choices", primary);
    let enabled = GuiShellAction::PlaceSketchTool(primary).dispatchable(state);
    let disabled = ui.begin_disabled(!enabled);
    ui.same_line_with_spacing(0., 2.);
    if widget_abi::button(ui, &format!("v{identity}"), [16., 18.]) && enabled {
        ui.open_popup(&identity);
    }
    drop(disabled);
    circular_tool_choices(ui, state, actions, &identity, [primary, alternate]);
}
fn circular_tool_choices(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    identity: &str,
    choices: [GuiSketchTool; 2],
) {
    let Some(_popup) = ui.begin_popup(identity) else {
        return;
    };
    for choice in choices {
        let action = GuiShellAction::PlaceSketchTool(choice);
        if !action.dispatchable(state) {
            ui.close_current_popup();
            return;
        }
        if ui.menu_item(choice.label()) {
            actions.push(action);
        }
    }
}
