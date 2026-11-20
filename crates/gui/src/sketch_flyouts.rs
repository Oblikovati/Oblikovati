// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction, GuiSketchTool, widget_abi};
use imgui::Ui;

pub(crate) fn sketch_tool_flyout_button(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    primary: GuiSketchTool,
    icons: &crate::icons::GuiRibbonIcons,
) {
    let alternate = match primary {
        GuiSketchTool::Line => Some(GuiSketchTool::MidpointLine),
        GuiSketchTool::Circle => Some(GuiSketchTool::ThreePointCircle),
        GuiSketchTool::CenterPointArc => Some(GuiSketchTool::ThreePointArc),
        GuiSketchTool::ThreePointRectangle => Some(GuiSketchTool::ThreePointCenterRectangle),
        _ => None,
    };
    let size = match icons.appearance {
        crate::ribbon_command_layout::RibbonAppearance::Normal if alternate.is_some() => [90., 18.],
        crate::ribbon_command_layout::RibbonAppearance::Normal => [108., 18.],
        _ => icons.appearance.cell(0).1,
    };
    icons.command_button(
        ui,
        primary.label(),
        GuiShellAction::PlaceSketchTool(primary),
        state,
        size,
        actions,
    );
    if let Some(alternate) = alternate {
        circular_tool_menu(ui, state, actions, primary, alternate, icons);
    }
}
fn circular_tool_menu(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    primary: GuiSketchTool,
    alternate: GuiSketchTool,
    icons: &crate::icons::GuiRibbonIcons,
) {
    let identity = format!("##{:?}-choices", primary);
    let enabled = GuiShellAction::PlaceSketchTool(primary).dispatchable(state);
    let disabled = ui.begin_disabled(!enabled);
    ui.same_line_with_spacing(0., 2. * icons.layout_factor());
    if widget_abi::button(ui, &format!("v{identity}"), icons.cell([16., 18.])) && enabled {
        ui.open_popup(&identity);
    }
    drop(disabled);
    let mut choices = vec![primary, alternate];
    if primary == GuiSketchTool::Circle {
        choices.push(GuiSketchTool::Ellipse);
    }
    circular_tool_choices(ui, state, actions, &identity, &choices);
}
fn circular_tool_choices(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    identity: &str,
    choices: &[GuiSketchTool],
) {
    let Some(_popup) = ui.begin_popup(identity) else {
        return;
    };
    for &choice in choices {
        let action = GuiShellAction::PlaceSketchTool(choice);
        if !action.dispatchable(state) {
            ui.close_current_popup();
            return;
        }
        if ui.menu_item(choice.menu_label()) {
            actions.push(action);
        }
    }
}
