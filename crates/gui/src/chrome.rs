// SPDX-License-Identifier: GPL-2.0-only
use crate::ribbon::{DocumentRibbon, RibbonFrame};
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};

pub(crate) fn ribbon_band(
    ui: &Ui,
    request: &GuiFrameRequest,
    ribbon: &mut DocumentRibbon,
    parameters: &mut crate::parameters::GuiParameterEditor,
    files: &mut crate::files::GuiDocumentFiles,
    theme_editor: &mut crate::theme_editor::GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) -> f32 {
    let width = request.extent[0] as f32 / request.framebuffer_scale[0];
    let height = 158. + ribbon.panels.scrollbar_height(ui);
    if let Some(_band) = begin_ribbon_band(ui, width, height) {
        ribbon_contents(
            ui,
            request,
            ribbon,
            parameters,
            files,
            theme_editor,
            actions,
        );
    }
    height
}
fn begin_ribbon_band(ui: &Ui, width: f32, height: f32) -> Option<imgui::WindowToken<'_>> {
    // #3798: chrome owns a fixed band; wheel input must never move its commands.
    crate::widget_abi::window_geometry(ui, [0., 0.], [width, height], Condition::Always);
    ui.window("##application-ribbon")
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .scroll_bar(false)
        .scrollable(false)
        .menu_bar(true)
        .begin()
}

fn ribbon_contents(
    ui: &Ui,
    request: &GuiFrameRequest,
    ribbon: &mut DocumentRibbon,
    parameters: &mut crate::parameters::GuiParameterEditor,
    files: &mut crate::files::GuiDocumentFiles,
    theme_editor: &mut crate::theme_editor::GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) {
    let frame = ribbon.prepare(&request.commands);
    ui.menu_bar(|| files.menu(ui, &request.commands, actions));
    crate::widget_abi::cursor_position(ui, [16., 35.]);
    quick_access(ui, &request.commands, actions);
    ui.same_line();
    ribbon_tabs(ui, ribbon, &frame);
    crate::widget_abi::cursor_position(ui, [16., 59.]);
    crate::ribbon_panels::draw_panels(
        ui,
        ribbon.active,
        &request.commands,
        parameters,
        theme_editor,
        &mut ribbon.panels,
        actions,
    );
    crate::theme_editor::panel(ui, request.commands.theme.as_ref(), theme_editor, actions);
    crate::widget_abi::cursor_position(ui, [8., ui.window_size()[1] - 20.]);
    ui.text(request.status.as_deref().unwrap_or("Ready"));
}
fn ribbon_tabs(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame) {
    let Some(_tabs) = ui.tab_bar("##document-ribbon-tabs") else {
        return;
    };
    for tab in &frame.tabs {
        let flags = if frame.select == Some(*tab) {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        if let Some(_item) = ui.tab_item_with_flags(tab.label(), None, flags) {
            ribbon.active = *tab;
        }
    }
}
fn quick_access(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    for (label, action) in [
        ("Save", GuiShellAction::Save),
        ("Undo", GuiShellAction::Undo),
        ("Redo", GuiShellAction::Redo),
        ("Home", GuiShellAction::Home),
    ] {
        command_button(ui, label, action, state, [0.; 2], actions);
        ui.same_line();
    }
}
pub(crate) fn sketch_tool_button(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    tool: crate::GuiSketchTool,
    size: [f32; 2],
) {
    command_button(
        ui,
        tool.label(),
        GuiShellAction::PlaceSketchTool(tool),
        state,
        size,
        actions,
    );
}
pub(crate) fn command_button(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    size: [f32; 2],
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = crate::widget_abi::button(ui, label, size);
    drop(disabled);
    if ui.is_item_hovered()
        && let Some(next) = action.next_label(state)
    {
        ui.tooltip_text(format!("{label} {next}"));
    }
    if clicked && action.dispatchable(state) {
        actions.push(action);
    }
}
