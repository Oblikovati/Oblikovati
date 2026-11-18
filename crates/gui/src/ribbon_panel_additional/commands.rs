// SPDX-License-Identifier: GPL-2.0-only
//! Render additional ribbon commands and their context menus.
use super::*;
pub(super) fn draw(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    display: &mut RibbonDisplayState,
) {
    let ui = context.ui;
    if builtin::supports(panel) {
        builtin::draw(
            ui,
            panel,
            context.state,
            &mut context.owners,
            context.actions,
            true,
        );
        return;
    }
    if panel == RibbonPanel::Create {
        create_commands(
            ui,
            context.state,
            context.owners.precise_input,
            context.owners.create_block,
            context.owners.icons,
            context.actions,
        );
        return;
    }
    if context.owners.icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal
        && let Some(entries) = additional_icon_entries(panel)
    {
        additional_icons(context, tab, panel, display, entries);
        return;
    }
    let labels = match panel {
        RibbonPanel::PartModify => &["Move Bodies", "Bend Part", "Copy Object"][..],
        RibbonPanel::PartSimplification => {
            &["Remove Details", "Fill Voids", "Define Envelopes"][..]
        }
        RibbonPanel::Constrain => &["Edit Coordinate System", "Constraint Inference Scope"][..],
        _ => return,
    };
    for &label in labels {
        let action = (label == "Move Bodies").then_some(GuiShellAction::OpenMoveBody);
        let disabled = ui.begin_disabled(
            action
                .as_ref()
                .is_none_or(|action| !action.dispatchable(context.state)),
        );
        if ui.button(label)
            && let Some(action) = action
        {
            context.actions.push(action);
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        unavailable_tooltip(ui, tab, panel, label, display);
    }
}
fn additional_icons(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    display: &mut RibbonDisplayState,
    entries: &'static [(&'static str, crate::icons::IconGlyph)],
) {
    let ui = context.ui;
    let icons = context.owners.icons;
    let origin = ui.cursor_pos();
    for (index, &(label, glyph)) in entries.iter().enumerate() {
        let (position, size) = icons.appearance.cell(index);
        icons.cursor(ui, origin, position);
        let action = (label == "Move Bodies").then_some(GuiShellAction::OpenMoveBody);
        let disabled = ui.begin_disabled(
            action
                .as_ref()
                .is_none_or(|action| !action.dispatchable(context.state)),
        );
        if icons.button(ui, label, glyph, size)
            && let Some(action) = action
        {
            context.actions.push(action);
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        unavailable_tooltip(ui, tab, panel, label, display);
    }
    icons.cursor(ui, origin, [0., icons.appearance.command_height()]);
}
fn additional_icon_entries(
    panel: RibbonPanel,
) -> Option<&'static [(&'static str, crate::icons::IconGlyph)]> {
    use crate::icons::IconGlyph as Glyph;
    Some(match panel {
        RibbonPanel::PartModify => &[
            ("Move Bodies", Glyph::MoveBodies),
            ("Bend Part", Glyph::BendPart),
            ("Copy Object", Glyph::Copy),
        ],
        RibbonPanel::PartSimplification => &[
            ("Remove Details", Glyph::Simplify),
            ("Fill Voids", Glyph::FillVoids),
            ("Define Envelopes", Glyph::Envelope),
        ],
        RibbonPanel::Constrain => &[
            ("Edit Coordinate System", Glyph::WorkCoordinateSystem),
            ("Constraint Inference Scope", Glyph::ConstraintScope),
        ],
        _ => return None,
    })
}
fn unavailable_tooltip(
    ui: &Ui,
    tab: RibbonTab,
    panel: RibbonPanel,
    label: &'static str,
    display: &mut RibbonDisplayState,
) {
    if !ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        return;
    }
    if ui.is_mouse_clicked(imgui::MouseButton::Right) {
        display.request_expansion_command_context(tab, panel, label);
    }
    if label == "Move Bodies" {
        ui.tooltip_text("Move Bodies\nMove a solid body using parameter-driven offsets.");
        return;
    }
    let help = if label == "Edit Coordinate System" {
        "Change the sketch origin and coordinate axes.\n"
    } else {
        ""
    };
    ui.tooltip_text(format!(
        "{label}\n{help}This command is not available in this build."
    ));
}
fn create_commands(
    ui: &Ui,
    state: &GuiCommandState,
    precise_input: &mut crate::sketch::precise_input::PreciseInputBar,
    create_block: &mut crate::sketch::create_block::CreateBlockDialog,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let size = if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        [0., 0.]
    } else {
        icons.appearance.cell(0).1
    };
    let origin = ui.cursor_pos();
    crate::sketch::create_block::ribbon_button(ui, state, create_block, actions, size, icons);
    let disabled = ui.begin_disabled(state.precise_target.is_none() || state.transaction_open);
    if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        precise_input.toggle(ui);
    } else {
        icons.cursor(ui, origin, icons.appearance.cell(1).0);
        precise_input.toggle_icon(ui, icons);
        icons.cursor(ui, origin, [0., icons.appearance.command_height()]);
    }
    drop(disabled);
}
