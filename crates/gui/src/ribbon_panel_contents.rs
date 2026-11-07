// SPDX-License-Identifier: GPL-2.0-only
//! Panel contents have one command path in the ribbon and its flyouts (#3849).
use crate::ribbon_panels::RibbonPanel;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

/// Beside the format toggles, which fill the panel's height.
const PRECISE_INPUT_COLUMN: f32 = 116.;

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
        RibbonPanel::Create => sketch_create_controls(ui, origin, state, owners.icons, actions),
        RibbonPanel::Constrain => {
            sketch_constraint_controls(ui, origin, state, owners.icons, actions)
        }
        RibbonPanel::Format => {
            crate::sketch_formats::sketch_format_controls(ui, origin, state, owners.icons, actions);
            owners.icons.cursor(ui, origin, [PRECISE_INPUT_COLUMN, 0.]);
            owners.precise_input.toggle(ui);
        }
        RibbonPanel::Parameters => return owners.parameters.toolbar(ui, state, owners.icons),
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
        RibbonPanel::DrawingModify => muted_commands(ui, origin, &["Break", "Crop"], NOT_PORTED),
        RibbonPanel::DrawingSketch => muted_commands(ui, origin, &["Start Sketch"], NOT_PORTED),
        RibbonPanel::DrawingSheets => muted_commands(ui, origin, &["New Sheet"], NOT_PORTED),
        RibbonPanel::AnnotateDimension => {
            muted_commands(ui, origin, &["General", "Baseline"], NOT_PORTED);
        }
        RibbonPanel::AnnotateText => muted_commands(ui, origin, &["Leader"], NOT_PORTED),
        RibbonPanel::AnnotateSymbols => {
            muted_commands(ui, origin, &["Surface", "Weld"], NOT_PORTED);
        }
        RibbonPanel::PartCreate => part_create_controls(ui, origin, state, owners, actions),
        RibbonPanel::PartModify => {
            muted_commands(ui, origin, &["Hole", "Fillet", "Chamfer"], NOT_PORTED);
        }
        RibbonPanel::PartPattern => {
            muted_commands(ui, origin, &["Rectangular", "Circular"], NOT_PORTED);
        }
        RibbonPanel::PartNotes => {
            muted_commands(ui, origin, &["General Note", "Leader Note"], NOT_PORTED);
        }
        RibbonPanel::PartWorkFeatures => {
            muted_commands(ui, origin, &["Plane", "Axis", "Point"], NOT_PORTED);
        }
        RibbonPanel::PresentationModel => {
            if crate::widget_abi::button(ui, "Insert Model", [108., 58.]) {
                owners.workspaces.request_insert();
            }
        }
        RibbonPanel::Sketch => start_sketch_command(ui, state, owners, actions),
        other => single_command(ui, other, state, owners.icons, actions),
    }
    false
}
/// Commands a document kind will offer: shown muted with the reason, never as a working button.
const NOT_PORTED: &str = "This command is not available in this build.";
fn muted_commands(ui: &Ui, origin: [f32; 2], labels: &[&str], reason: &str) {
    for (index, label) in labels.iter().enumerate() {
        let column = (index / 2) as f32 * 116.;
        let row = (index % 2) as f32 * 30.;
        crate::widget_abi::cursor_position(ui, [origin[0] + column, origin[1] + row]);
        {
            let _muted = ui.begin_disabled(true);
            crate::widget_abi::button(ui, label, [108., 26.]);
        }
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text(reason);
        }
    }
}
/// Extrude starts a sketch when needed, or opens the profile dialog.
fn part_create_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    use crate::application::quick_access_toolbar::{EXTRUDE, extrude_can_start};
    crate::widget_abi::cursor_position(ui, origin);
    {
        let _muted = ui.begin_disabled(!extrude_can_start(state));
        if crate::widget_abi::button(ui, EXTRUDE, [108., 26.]) {
            actions.push(crate::extrude_dialog::start_action(state));
        }
    }
    if !extrude_can_start(state)
        && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
    {
        ui.tooltip_text(NOT_PORTED);
    }
    owners.quick_access.ribbon_command_menu(ui, EXTRUDE, state);
    crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 30.]);
    {
        let _muted = ui.begin_disabled(true);
        crate::widget_abi::button(ui, "Revolve", [108., 26.]);
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(NOT_PORTED);
    }
    owners
        .quick_access
        .ribbon_command_menu(ui, "Revolve", state);
}
/// Stays lit while the origin planes wait for a choice.
fn start_sketch_command(
    ui: &Ui,
    state: &GuiCommandState,
    owners: &crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let lit = ui.style_color(imgui::StyleColor::ButtonActive);
    let _highlight = owners
        .plane_selection_active
        .then(|| ui.push_style_color(imgui::StyleColor::Button, lit));
    single_command(ui, RibbonPanel::Sketch, state, owners.icons, actions);
}
fn single_command(
    ui: &Ui,
    panel: RibbonPanel,
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    if let Some((label, action)) = panel.command() {
        icons.command_button(ui, label, action, state, [108., 58.], actions);
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
    crate::widget_abi::cursor_position(ui, [origin[0] + 116., origin[1]]);
    crate::chrome::command_button(
        ui,
        "New",
        GuiShellAction::NewDocumentWindow,
        state,
        [108., 26.],
        actions,
    );
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
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    for (choice, position) in CONSTRAINT_CONTROLS {
        icons.cursor(ui, origin, position);
        icons.command_button(
            ui,
            choice.label(),
            GuiShellAction::ConstrainSketch(choice),
            state,
            [108., 18.],
            actions,
        );
    }
    icons.cursor(ui, origin, [116., 0.]);
    icons.command_button(
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
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::widget_abi::cursor_position(ui, origin);
    let rectangle = crate::GuiSketchTool::Rectangle;
    icons.command_button(
        ui,
        rectangle.label(),
        GuiShellAction::PlaceSketchTool(rectangle),
        state,
        [108., 58.],
        actions,
    );
    let secondary = crate::sketch_tools::SECONDARY_SKETCH_TOOLS;
    for (index, tool) in secondary.into_iter().enumerate() {
        icons.cursor(
            ui,
            origin,
            [124. + (index % 2) as f32 * 116., (index / 2) as f32 * 20.],
        );
        crate::sketch_flyouts::circular_tool_button(ui, state, actions, tool, icons);
    }
    icons.cursor(ui, origin, [124., 60.]);
    icons.command_button(
        ui,
        "Line",
        GuiShellAction::PlaceSketchTool(crate::GuiSketchTool::Line),
        state,
        [108., 18.],
        actions,
    );
}
