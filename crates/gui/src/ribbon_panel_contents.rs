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
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    crate::widget_abi::cursor_position(ui, origin);
    match panel {
        RibbonPanel::Create => sketch_create_controls(ui, origin, state, actions),
        RibbonPanel::Constrain => sketch_constraint_controls(ui, origin, state, actions),
        RibbonPanel::Format => {
            crate::sketch_formats::sketch_format_controls(ui, origin, state, actions)
        }
        RibbonPanel::Parameters => return owners.parameters.toolbar(ui, state),
        RibbonPanel::Appearance => return appearance_controls(ui, state, owners.editor, actions),
        RibbonPanel::Visibility => visibility_controls(ui, state, actions),
        RibbonPanel::Windows => windows_controls(ui, origin, state, owners, actions),
        RibbonPanel::Component => muted_commands(ui, origin, &["Place", "Create"], NOT_PORTED),
        RibbonPanel::SheetMetalTools => {
            muted_commands(ui, origin, &["Flange", "Flat Pattern"], NOT_PORTED);
        }
        RibbonPanel::DrawingViews => {
            muted_commands(ui, origin, &["Base View", "Projected View"], NOT_PORTED);
        }
        RibbonPanel::PresentationModel => {
            if crate::widget_abi::button(ui, "Insert Model", [108., 58.]) {
                owners.workspaces.request_insert();
            }
        }
        other => single_command(ui, other, state, actions),
    }
    false
}
/// Commands a document kind will offer: shown muted with the reason, never as a working button.
const NOT_PORTED: &str = "This command is not available in this build.";
fn muted_commands(ui: &Ui, origin: [f32; 2], labels: &[&str], reason: &str) {
    for (row, label) in labels.iter().enumerate() {
        crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + row as f32 * 30.]);
        {
            let _muted = ui.begin_disabled(true);
            crate::widget_abi::button(ui, label, [108., 26.]);
        }
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text(reason);
        }
    }
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
fn windows_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    owners.workspace.user_interface_menu(ui, [108., 26.]);
    crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 30.]);
    owners
        .workspace
        .switcher
        .menu(ui, [108., 26.], state, actions);
}
/// Show or hide every sketch at once through the owner's per-sketch command.
fn visibility_controls(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let sketches = state
        .browser
        .as_ref()
        .map_or(&[][..], |browser| &browser.sketches);
    let hide = sketches.iter().any(|row| row.visible);
    let disabled = ui.begin_disabled(sketches.is_empty() || state.transaction_open);
    let label = if hide {
        "Hide Sketches"
    } else {
        "Show Sketches"
    };
    let pressed = crate::widget_abi::button(ui, label, [108., 26.]);
    drop(disabled);
    let Some(document) = state.document.filter(|_| pressed) else {
        return;
    };
    actions.extend(
        sketches
            .iter()
            .map(|row| GuiShellAction::SetSketchVisibility {
                document,
                sketch: row.id,
                visible: !hide,
            }),
    );
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
