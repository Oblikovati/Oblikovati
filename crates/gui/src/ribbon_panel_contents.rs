// SPDX-License-Identifier: GPL-2.0-only
//! Panel contents have one command path in the ribbon and its flyouts (#3849).
use crate::ribbon_panels::RibbonPanel;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

pub(crate) fn draw_panel_content(
    ui: &Ui,
    panel: RibbonPanel,
    origin: [f32; 2],
    state: &GuiCommandState,
    parameters: &mut crate::parameters::GuiParameterEditor,
    editor: &mut crate::theme_editor::GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    crate::widget_abi::cursor_position(ui, origin);
    match panel {
        RibbonPanel::Create => sketch_create_controls(ui, origin, state, actions),
        RibbonPanel::Constrain => sketch_constraint_controls(ui, origin, state, actions),
        RibbonPanel::Format => {
            crate::sketch_formats::sketch_format_controls(ui, origin, state, actions)
        }
        RibbonPanel::Parameters => return parameters.toolbar(ui, state),
        RibbonPanel::Appearance => return appearance_controls(ui, state, editor, actions),
        other => single_command(ui, other, state, actions),
    }
    false
}
fn single_command(
    ui: &Ui,
    panel: RibbonPanel,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    if let Some((label, action)) = panel.command() {
        crate::chrome::command_button(ui, label, action, state, [108., 58.], actions);
    }
}
fn appearance_controls(
    ui: &Ui,
    state: &GuiCommandState,
    editor: &mut crate::theme_editor::GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    if let Some(theme) = &state.theme {
        theme.controls(ui, actions);
    }
    if let Some(palette) = &state.color_scheme {
        palette.controls(ui, actions);
    }
    state.theme.is_some() && crate::theme_editor::launcher(ui, editor)
}
use crate::GuiSketchConstraint::{
    Coincident, Collinear, Concentric, Equal, Fix, Horizontal, Parallel, Perpendicular, Tangent,
    Vertical,
};
const CONSTRAINT_CONTROLS: [(crate::GuiSketchConstraint, [f32; 2]); 10] = [
    (Horizontal, [0.0, 0.0]),
    (Vertical, [0.0, 20.0]),
    (Coincident, [0.0, 40.0]),
    (Fix, [116.0, 60.0]),
    (Parallel, [232.0, 0.0]),
    (Perpendicular, [232.0, 20.0]),
    (Collinear, [232.0, 40.0]),
    (Equal, [348.0, 0.0]),
    (Concentric, [348.0, 20.0]),
    (Tangent, [348.0, 40.0]),
];
fn sketch_constraint_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    for (choice, position) in CONSTRAINT_CONTROLS {
        crate::widget_abi::cursor_position(ui, [origin[0] + position[0], origin[1] + position[1]]);
        crate::chrome::command_button(
            ui,
            choice.label(),
            GuiShellAction::ConstrainSketch(choice),
            state,
            [108., 18.],
            actions,
        );
    }
    crate::widget_abi::cursor_position(ui, [origin[0] + 116., origin[1]]);
    crate::chrome::command_button(
        ui,
        "Dimension",
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
        state,
        [108., 58.],
        actions,
    );
}
fn sketch_create_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::widget_abi::cursor_position(ui, origin);
    let rectangle = crate::GuiSketchTool::Rectangle;
    crate::chrome::sketch_tool_button(ui, state, actions, rectangle, [108., 58.]);
    let secondary = crate::sketch_tools::SECONDARY_SKETCH_TOOLS;
    for (index, tool) in secondary.into_iter().enumerate() {
        crate::widget_abi::cursor_position(
            ui,
            [
                origin[0] + 124. + (index % 2) as f32 * 116.,
                origin[1] + (index / 2) as f32 * 20.,
            ],
        );
        crate::sketch_flyouts::circular_tool_button(ui, state, actions, tool);
    }
    crate::widget_abi::cursor_position(ui, [origin[0] + 124., origin[1] + 60.]);
    crate::chrome::sketch_tool_button(ui, state, actions, crate::GuiSketchTool::Line, [108., 18.]);
}
