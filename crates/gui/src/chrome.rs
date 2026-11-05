// SPDX-License-Identifier: GPL-2.0-only
use crate::ribbon::{DocumentRibbon, RibbonFrame, RibbonTab};
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};

pub(crate) fn ribbon_band(
    ui: &Ui,
    request: &GuiFrameRequest,
    ribbon: &mut DocumentRibbon,
    parameters: &mut crate::parameters::GuiParameterEditor,
    files: &mut crate::files::GuiDocumentFiles,
    actions: &mut Vec<GuiShellAction>,
) {
    let width = request.extent[0] as f32 / request.framebuffer_scale[0];
    let frame = ribbon.prepare(&request.commands);
    if let Some(_band) = begin_ribbon_band(ui, width) {
        ribbon_contents(ui, request, ribbon, &frame, parameters, files, actions);
    }
}
fn begin_ribbon_band(ui: &Ui, width: f32) -> Option<imgui::WindowToken<'_>> {
    // #3798: chrome owns a fixed band; wheel input must never move its commands.
    ui.window("##application-ribbon")
        .position([0., 0.], Condition::Always)
        .size([width, 158.], Condition::Always)
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
    frame: &RibbonFrame,
    parameters: &mut crate::parameters::GuiParameterEditor,
    files: &mut crate::files::GuiDocumentFiles,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.menu_bar(|| files.menu(ui, &request.commands, actions));
    ui.set_cursor_pos([16., 35.]);
    quick_access(ui, &request.commands, actions);
    ui.same_line();
    ribbon_tabs(ui, ribbon, frame);
    ui.set_cursor_pos([16., 59.]);
    tab_panels(ui, ribbon.active, &request.commands, parameters, actions);
    ui.set_cursor_pos([8., 138.]);
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
fn tab_panels(
    ui: &Ui,
    tab: RibbonTab,
    state: &GuiCommandState,
    parameters: &mut crate::parameters::GuiParameterEditor,
    actions: &mut Vec<GuiShellAction>,
) {
    let command = match tab {
        RibbonTab::GetStarted => ("Launch", "New Part", GuiShellAction::NewPart),
        RibbonTab::CreateModify => ("Sketch", "Sketch XY", GuiShellAction::StartSketchXY),
        RibbonTab::View => ("Navigate", "Home", GuiShellAction::Home),
        RibbonTab::Sketch => return sketch_panels(ui, state, actions),
        RibbonTab::Manage => return parameter_panel(ui, state, parameters),
    };
    command_panel(ui, state, actions, command);
}

fn sketch_panels(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let create = ("Create", "Rectangle", GuiShellAction::Rectangle);
    command_panel(ui, state, actions, create);
    ui.same_line();
    ui.set_cursor_pos([140., 59.]);
    let exit = ("Exit", "Finish Sketch", GuiShellAction::FinishSketch);
    command_panel(ui, state, actions, exit);
}
type RibbonCommandSpec = (&'static str, &'static str, GuiShellAction);
fn command_panel(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    command: RibbonCommandSpec,
) {
    let (panel, label, action) = command;
    let _identity = ui.push_id(panel);
    ui.group(|| {
        command_button(ui, label, action, state, [108., 58.], actions);
        ui.spacing();
        ui.text(panel);
    });
}
fn parameter_panel(
    ui: &Ui,
    state: &GuiCommandState,
    parameters: &mut crate::parameters::GuiParameterEditor,
) {
    ui.group(|| {
        parameters.toolbar(ui, state);
        ui.spacing();
        ui.text("Parameters");
    });
}
fn command_button(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    size: [f32; 2],
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = ui.button_with_size(label, size);
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
