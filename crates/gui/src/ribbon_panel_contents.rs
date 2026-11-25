// SPDX-License-Identifier: GPL-2.0-only
//! Panel contents have one command path in the ribbon and its flyouts (#3849).
use crate::ribbon_panels::RibbonPanel;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
mod appearance_other;
mod appearance_sketch;
mod appearance_tools;
pub(super) use appearance_tools::highlight_new;
mod appearance_view;
mod muted;
pub(crate) use muted::{
    FLOATING_NOTES_HEIGHT, FLOATING_NOTES_WIDTH, compact_modify_commands, compact_modify_width,
    floating_notes,
};
mod sketch;
use muted::{muted_icon_commands, part_modify_controls};

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
    if appearance_other::draw(ui, panel, origin, state, owners, actions) {
        return false;
    }
    match panel {
        RibbonPanel::Modify => {
            if owners.icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
                appearance_sketch::modify(ui, origin, state, owners.icons, actions);
                return false;
            }
            let (_, size) = owners.icons.appearance.cell(0);
            let start = GuiShellAction::SketchMove(crate::GuiSketchMoveAction::Start);
            crate::chrome::command_control(ui, "Move", start, state, actions, || {
                owners
                    .icons
                    .button(ui, "Move", crate::icons::IconGlyph::Move, size)
            });
            let (position, size) = owners.icons.appearance.cell(1);
            owners.icons.cursor(ui, origin, position);
            let delete = GuiShellAction::DeleteSketchSelection;
            crate::chrome::command_control(ui, "Delete", delete, state, actions, || {
                owners
                    .icons
                    .button(ui, "Delete", crate::icons::IconGlyph::Delete, size)
            });
        }
        RibbonPanel::SketchPattern => {
            let opened = crate::sketch::rectangular_pattern::ribbon_button(
                ui,
                state,
                owners.sketch_pattern,
                owners.icons,
            );
            let (position, size) = owners.icons.appearance.cell(1);
            owners.icons.cursor(ui, origin, position);
            let unavailable = ui.begin_disabled(true);
            owners.icons.button(
                ui,
                "Circular",
                crate::icons::IconGlyph::CircularPattern,
                size,
            );
            drop(unavailable);
            return opened;
        }
        RibbonPanel::Insert => muted_icon_commands(
            ui,
            origin,
            owners.icons,
            &[
                ("Image", Some(crate::icons::IconGlyph::InsertImage)),
                ("Points", Some(crate::icons::IconGlyph::Point)),
            ],
            false,
        ),
        RibbonPanel::ManageInsert => muted::manage_insert_commands(ui, origin, owners.icons),
        RibbonPanel::Create => {
            sketch::sketch_create_controls(ui, origin, state, owners.icons, actions)
        }
        RibbonPanel::Constrain => {
            sketch::sketch_constraint_controls(ui, origin, state, owners.icons, actions)
        }
        RibbonPanel::Layout => {
            muted_icon_commands(
                ui,
                origin,
                owners.icons,
                &[
                    ("Make Part", Some(crate::icons::IconGlyph::NewPart)),
                    (
                        "Make Components",
                        Some(crate::icons::IconGlyph::NewAssembly),
                    ),
                ],
                false,
            );
            let (position, size) = owners.icons.appearance.cell(2);
            owners.icons.cursor(ui, origin, position);
            crate::sketch::create_block::ribbon_button(
                ui,
                state,
                owners.create_block,
                actions,
                crate::sketch::create_block::CreateBlockButton::Ribbon(size),
                owners.icons,
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
            let position = if owners.icons.appearance
                == crate::ribbon_command_layout::RibbonAppearance::Normal
            {
                [PRECISE_INPUT_COLUMN, 0.]
            } else {
                owners.icons.appearance.cell(3).0
            };
            owners.icons.cursor(ui, origin, position);
            let disabled = ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Sketch);
            if owners.icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
                owners
                    .precise_input
                    .toggle(ui, owners.icons.cell([108., 20.]));
            } else {
                owners.precise_input.toggle_icon(ui, owners.icons);
            }
            drop(disabled);
        }
        RibbonPanel::Parameters => {
            let clicked = owners.parameters.toolbar(ui, state, owners.icons);
            if clicked {
                owners.workspace.tool_visibility.parameters = owners.parameters.is_open();
            }
            return clicked;
        }
        RibbonPanel::ToolOptions => {
            return appearance_tools::options(ui, origin, state, owners);
        }
        RibbonPanel::ToolAssets => {
            crate::ribbon_part_assets::draw(ui, origin, state, owners.icons, actions)
        }
        RibbonPanel::ToolClipboard | RibbonPanel::ToolMeasure => {
            crate::ribbon_panels::builtin::draw(ui, panel, state, owners, actions, false);
        }
        // The captured Tutorials panel contains no commands. Its title follows Show Panels.
        RibbonPanel::ToolTutorials => {}
        RibbonPanel::ToolTeamWeb => return owners.team_web.ribbon_button(ui, owners.icons),
        RibbonPanel::ToolFind => appearance_tools::find(ui, state, owners),
        RibbonPanel::Appearance => {
            if owners.icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
                return appearance_view::theme(
                    ui,
                    origin,
                    state,
                    owners.editor,
                    owners.icons,
                    actions,
                );
            }
            return appearance_controls(ui, state, owners.editor, owners.icons, actions);
        }
        RibbonPanel::Visibility => sketch::visibility_controls(ui, state, owners.icons, actions),
        RibbonPanel::Windows => appearance_view::windows(ui, origin, state, owners, actions),
        RibbonPanel::PartCreate => part_create_controls(ui, origin, state, owners, actions),
        RibbonPanel::PartModify => {
            part_modify_controls(ui, origin, state, actions, owners);
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
            muted::muted_command(
                ui,
                "Simplify",
                NOT_PORTED,
                owners.icons,
                Some(crate::icons::IconGlyph::Simplify),
                false,
                owners.icons.appearance.single_cell(),
            );
        }
        RibbonPanel::PartWorkFeatures => {
            use crate::icons::IconGlyph;
            let (position, _) = owners.icons.appearance.cell(0);
            owners.icons.cursor(ui, origin, position);
            crate::ribbon_panels::builtin::draw(ui, panel, state, owners, actions, false);
            let plane_cells =
                crate::ribbon_panels::builtin::plane_main_cells(state, owners.customization);
            for (index, (label, icon)) in [
                ("Axis", IconGlyph::WorkAxis),
                ("Point", IconGlyph::WorkPoint),
                ("UCS", IconGlyph::WorkCoordinateSystem),
            ]
            .into_iter()
            .enumerate()
            {
                let (position, size) = owners.icons.appearance.cell(index + plane_cells);
                owners.icons.cursor(ui, origin, position);
                muted::muted_command(ui, label, NOT_PORTED, owners.icons, Some(icon), false, size);
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
    let (position, size) = owners.icons.appearance.cell(0);
    owners.icons.cursor(ui, origin, position);
    {
        let _muted = ui.begin_disabled(!extrude_can_start(state));
        if owners
            .icons
            .button(ui, EXTRUDE, crate::icons::IconGlyph::Extrude, size)
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
    owners
        .quick_access
        .ribbon_command_menu(ui, EXTRUDE, true, state);
    let (position, size) = owners.icons.appearance.cell(1);
    owners.icons.cursor(ui, origin, position);
    {
        let _muted = ui.begin_disabled(true);
        owners
            .icons
            .button(ui, "Revolve", crate::icons::IconGlyph::Revolve, size);
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(NOT_PORTED);
    }
    owners
        .quick_access
        .ribbon_command_menu(ui, "Revolve", false, state);
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
        icons.command_button(
            ui,
            label,
            action,
            state,
            icons.appearance.single_cell(),
            actions,
        );
    }
}
fn appearance_controls(
    ui: &Ui,
    state: &GuiCommandState,
    editor: &mut crate::theme_editor::GuiThemeEditor,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    if let Some(theme) = &state.theme {
        theme.controls(ui, actions);
    }
    if let Some(palette) = &state.color_scheme {
        palette.controls(ui, actions);
    }
    state.theme.is_some() && crate::theme_editor::launcher(ui, editor, icons)
}
