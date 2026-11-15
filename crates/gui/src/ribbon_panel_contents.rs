// SPDX-License-Identifier: GPL-2.0-only
//! Panel contents have one command path in the ribbon and its flyouts (#3849).
use crate::ribbon_panels::RibbonPanel;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
mod muted;
use muted::{muted_commands, muted_icon_commands, part_modify_controls};

/// Beside the format toggles, which fill the panel's height.
const PRECISE_INPUT_COLUMN: f32 = 116.;

pub(crate) fn draw_panel_content(
    ui: &Ui,
    panel: RibbonPanel,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
    additional_allowed: bool,
) -> bool {
    crate::widget_abi::cursor_position(ui, origin);
    match panel {
        RibbonPanel::Modify => {
            let start = GuiShellAction::SketchMove(crate::GuiSketchMoveAction::Start);
            crate::chrome::command_control(ui, "Move", start, state, actions, || {
                owners
                    .icons
                    .button(ui, "Move", crate::icons::IconGlyph::Move, [108., 26.])
            });
            owners.icons.cursor(ui, origin, [0., 30.]);
            let delete = GuiShellAction::DeleteSketchSelection;
            crate::chrome::command_control(ui, "Delete", delete, state, actions, || {
                crate::widget_abi::button(ui, "Delete", [108., 26.])
            });
        }
        RibbonPanel::SketchPattern => {
            let opened =
                crate::sketch::rectangular_pattern::ribbon_button(ui, state, owners.sketch_pattern);
            owners.icons.cursor(ui, origin, [0., 30.]);
            let unavailable = ui.begin_disabled(true);
            crate::widget_abi::button(ui, "Circular", [108., 26.]);
            drop(unavailable);
            return opened;
        }
        RibbonPanel::Insert => {
            muted_commands(ui, origin, owners.icons, &["Image", "Points"], NOT_PORTED)
        }
        RibbonPanel::Create => sketch_create_controls(ui, origin, state, owners.icons, actions),
        RibbonPanel::Constrain => {
            sketch_constraint_controls(ui, origin, state, owners.icons, actions)
        }
        RibbonPanel::Layout => {
            muted_commands(
                ui,
                origin,
                owners.icons,
                &["Make Part", "Make Components"],
                NOT_PORTED,
            );
            owners.icons.cursor(ui, origin, [116., 0.]);
            crate::sketch::create_block::ribbon_button(
                ui,
                state,
                owners.create_block,
                actions,
                [108., 26.],
            );
        }
        RibbonPanel::UserCommands => {
            let right = owners.customization.draw_panel(
                ui,
                owners.active_tab,
                additional_allowed.then_some(false),
                state,
                owners.icons,
                actions,
            );
            owners.quick_access.command_took_right_click |= right;
        }
        RibbonPanel::Format => {
            crate::sketch_formats::sketch_format_controls(ui, origin, state, owners.icons, actions);
            owners.icons.cursor(ui, origin, [PRECISE_INPUT_COLUMN, 0.]);
            let disabled = ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Sketch);
            owners.precise_input.toggle(ui);
            drop(disabled);
        }
        RibbonPanel::Parameters => return owners.parameters.toolbar(ui, state, owners.icons),
        RibbonPanel::ToolOptions => {
            owners.icons.cursor(ui, origin, [156., 0.]);
            let disabled = ui.begin_disabled(
                state.document_type != Some(oblikovati_api::types::DOCUMENT_PART)
                    || owners.home_active,
            );
            if crate::widget_abi::button(ui, "Customize", [108., 26.]) {
                owners.customization.open(owners.active_tab);
            }
            drop(disabled);
            crate::widget_abi::cursor_position(ui, origin);
            let disabled = ui.begin_disabled(state.ui_scale.is_none());
            let application = crate::widget_abi::button(ui, "Application Options", [150., 26.]);
            if application {
                owners.preferences.request();
            }
            drop(disabled);
            crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 30.]);
            let disabled = ui.begin_disabled(
                state.document_units.is_none() || state.transaction_open || owners.home_active,
            );
            let document = crate::widget_abi::button(ui, "Document Settings", [150., 26.]);
            drop(disabled);
            if document && let Some(id) = state.document {
                owners.preferences.request_document(id);
            }
            return application || document;
        }
        RibbonPanel::ToolAssets => crate::ribbon_part_assets::draw(ui, origin, state, actions),
        RibbonPanel::ToolClipboard | RibbonPanel::ToolMeasure => {
            crate::ribbon_panels::builtin::draw(ui, panel, state, owners, actions, false);
        }
        RibbonPanel::ToolFind => {
            let disabled = ui.begin_disabled(
                owners.home_active
                    || state
                        .browser
                        .as_ref()
                        .is_none_or(|browser| browser.kind == crate::GuiBrowserKind::Drawing),
            );
            if crate::widget_abi::button(ui, "Find", [108., 58.]) {
                owners.workspace.request_find();
            }
            drop(disabled);
        }
        RibbonPanel::Appearance => return appearance_controls(ui, state, owners.editor, actions),
        RibbonPanel::Visibility => visibility_controls(ui, state, actions),
        RibbonPanel::Windows => windows_controls(ui, origin, state, owners, actions),
        RibbonPanel::Component => {
            owners.icons.cursor(ui, origin, [0., 0.]);
            if state.active_express() {
                if let Some(id) = state.document {
                    crate::chrome::command_control(
                        ui,
                        "Load Full",
                        GuiShellAction::LoadFullDocument(id),
                        state,
                        actions,
                        || {
                            crate::widget_abi::button(
                                ui,
                                "Load Full",
                                owners.icons.cell([108., 58.]),
                            )
                        },
                    );
                }
                return false;
            }
            let disabled = ui.begin_disabled(owners.home_active || state.transaction_open);
            if crate::widget_abi::button(ui, "Place", owners.icons.cell([108., 26.])) {
                owners.workspaces.request_assembly_place();
            }
            drop(disabled);
            owners.icons.cursor(ui, origin, [0., 30.]);
            let disabled = ui.begin_disabled(true);
            crate::widget_abi::button(ui, "Create", owners.icons.cell([108., 26.]));
            drop(disabled);
        }
        RibbonPanel::SheetMetalTools => {
            muted_commands(
                ui,
                origin,
                owners.icons,
                &["Flange", "Flat Pattern"],
                NOT_PORTED,
            );
        }
        RibbonPanel::DrawingViews => {
            let _disabled =
                ui.begin_disabled(state.transaction_open || state.drawing_updates_deferred);
            if crate::widget_abi::button(ui, "Base View", owners.icons.cell([108., 26.])) {
                owners.workspaces.request_base_view();
            }
            drop(_disabled);
            muted_commands(
                ui,
                [origin[0], origin[1] + owners.icons.cell([0., 30.])[1]],
                owners.icons,
                &["Projected View"],
                NOT_PORTED,
            );
        }
        RibbonPanel::DrawingModify => {
            muted_commands(ui, origin, owners.icons, &["Break", "Crop"], NOT_PORTED)
        }
        RibbonPanel::DrawingSketch => {
            muted_commands(ui, origin, owners.icons, &["Start Sketch"], NOT_PORTED)
        }
        RibbonPanel::DrawingSheets => {
            muted_commands(ui, origin, owners.icons, &["New Sheet"], NOT_PORTED)
        }
        RibbonPanel::AnnotateDimension => {
            muted_commands(
                ui,
                origin,
                owners.icons,
                &["General", "Baseline"],
                NOT_PORTED,
            );
        }
        RibbonPanel::AnnotateText => {
            muted_commands(ui, origin, owners.icons, &["Leader"], NOT_PORTED)
        }
        RibbonPanel::AnnotateSymbols => {
            muted_commands(ui, origin, owners.icons, &["Surface", "Weld"], NOT_PORTED);
        }
        RibbonPanel::PartCreate => part_create_controls(ui, origin, state, owners, actions),
        RibbonPanel::PartModify => {
            part_modify_controls(ui, origin, owners.icons, owners.hole_tip);
        }
        RibbonPanel::PartPattern => {
            use crate::icons::IconGlyph;
            muted_icon_commands(
                ui,
                origin,
                owners.icons,
                &[
                    ("Rectangular", Some(IconGlyph::RectangularPattern)),
                    ("Circular", Some(IconGlyph::CircularPattern)),
                ],
                false,
            );
        }
        RibbonPanel::PartSimplification => {
            muted_icon_commands(
                ui,
                origin,
                owners.icons,
                &[("Simplify", Some(crate::icons::IconGlyph::Simplify))],
                false,
            );
        }
        RibbonPanel::PartNotes => {
            muted_commands(
                ui,
                origin,
                owners.icons,
                &["General Note", "Leader Note"],
                NOT_PORTED,
            );
        }
        RibbonPanel::PartWorkFeatures => {
            use crate::icons::IconGlyph;
            muted_icon_commands(
                ui,
                origin,
                owners.icons,
                &[
                    ("Plane", Some(IconGlyph::WorkPlane)),
                    ("Axis", Some(IconGlyph::WorkAxis)),
                    ("Point", Some(IconGlyph::WorkPoint)),
                    ("UCS", None),
                ],
                false,
            );
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
        if owners
            .icons
            .button(ui, EXTRUDE, crate::icons::IconGlyph::Extrude, [108., 26.])
        {
            actions.push(crate::extrude_dialog::start_action(state));
        }
        if owners.extrude_tip {
            crate::ribbon_key_tips::badge(ui, "EX");
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
        owners
            .icons
            .button(ui, "Revolve", crate::icons::IconGlyph::Revolve, [108., 26.]);
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
    if owners.sketch_tip {
        crate::ribbon_key_tips::badge(ui, "SK");
    }
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
    if ui.is_item_hovered() {
        ui.tooltip_text("Draw connected line segments. For arcs, choose Center Arc.");
    }
    icons.cursor(ui, origin, [240., 60.]);
    let disabled = ui.begin_disabled(true);
    crate::widget_abi::button(ui, "Project Geometry", icons.cell([150., 18.]));
    drop(disabled);
}
