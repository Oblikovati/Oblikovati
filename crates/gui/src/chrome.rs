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
    theme_editor: &mut crate::theme_editor::GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) {
    let width = request.extent[0] as f32 / request.framebuffer_scale[0];
    if let Some(_band) = begin_ribbon_band(ui, width) {
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
}
fn begin_ribbon_band(ui: &Ui, width: f32) -> Option<imgui::WindowToken<'_>> {
    // #3798: chrome owns a fixed band; wheel input must never move its commands.
    crate::widget_abi::window_geometry(ui, [0., 0.], [width, 158.], Condition::Always);
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
    tab_panels(ui, ribbon.active, &request.commands, parameters, actions);
    crate::theme_editor::panel(ui, request.commands.theme.as_ref(), theme_editor, actions);
    crate::widget_abi::cursor_position(ui, [8., 138.]);
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
    if tab == RibbonTab::View {
        appearance_panel(ui, state, actions);
    }
}

fn sketch_panels(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    ui.group(|| sketch_create_controls(ui, state, actions));
    ui.same_line();
    crate::widget_abi::cursor_position(ui, [388., 59.]);
    let exit = ("Exit", "Finish Sketch", GuiShellAction::FinishSketch);
    command_panel(ui, state, actions, exit);
    sketch_constraint_controls(ui, state, actions);
    crate::widget_abi::cursor_position(ui, [620., 59.]);
    command_button(
        ui,
        "Dimension",
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
        state,
        [108., 58.],
        actions,
    );
}
fn sketch_constraint_controls(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    use crate::GuiSketchConstraint::{Coincident, Fix, Horizontal, Vertical};
    for (choice, position) in [
        (Horizontal, [504., 59.]),
        (Vertical, [504., 79.]),
        (Coincident, [504., 99.]),
        (Fix, [620., 119.]),
    ] {
        crate::widget_abi::cursor_position(ui, position);
        command_button(
            ui,
            choice.label(),
            GuiShellAction::ConstrainSketch(choice),
            state,
            [108., 18.],
            actions,
        );
    }
    crate::widget_abi::cursor_position(ui, [504., 125.]);
    ui.text("Constrain");
}
fn sketch_create_controls(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let rectangle = crate::GuiSketchTool::Rectangle;
    sketch_tool_button(ui, state, actions, rectangle, [108., 58.]);
    let secondary = crate::sketch_tools::SECONDARY_SKETCH_TOOLS;
    for (index, tool) in secondary.into_iter().enumerate() {
        crate::widget_abi::cursor_position(
            ui,
            [
                140. + (index % 2) as f32 * 116.,
                59. + (index / 2) as f32 * 20.,
            ],
        );
        sketch_tool_button(ui, state, actions, tool, [108., 18.]);
    }
    crate::widget_abi::cursor_position(ui, [16., 125.]);
    ui.text("Create");
}
fn sketch_tool_button(
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

fn appearance_panel(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    if state.theme.is_none() && state.color_scheme.is_none() {
        return;
    }
    ui.same_line();
    ui.group(|| {
        if let Some(theme) = &state.theme {
            theme.controls(ui, actions);
        }
        if let Some(palette) = &state.color_scheme {
            palette.controls(ui, actions);
        }
        if state.theme.is_some() {
            crate::theme_editor::launcher(ui);
        }
        ui.text("Appearance");
    });
}
