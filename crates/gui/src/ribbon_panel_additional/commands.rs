// SPDX-License-Identifier: GPL-2.0-only
//! Render additional ribbon commands and their context menus.
use super::*;
#[path = "coordinate_restriction.rs"]
mod coordinate_restriction;
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
        RibbonPanel::PartModify => &["Move Bodies", "Bend Part", "Copy Object", "Shell"][..],
        RibbonPanel::PartSimplification => {
            &["Remove Details", "Fill Voids", "Define Envelopes"][..]
        }
        RibbonPanel::Constrain => &["Edit Coordinate System", "Constraint Inference Scope"][..],
        _ => return,
    };
    for &label in labels {
        if label == "Move Bodies" && crate::move_body_ribbon::is_main(context.owners.customization)
        {
            continue;
        }
        if label == "Shell" && !crate::shell_ribbon::is_expanded(context.owners.customization) {
            continue;
        }
        let action = crate::shell_ribbon::additional_action(label);
        let coordinate = coordinate_restriction::ready(label, context.state);
        let disabled = ui.begin_disabled(
            !coordinate
                && action
                    .as_ref()
                    .is_none_or(|action| !action.dispatchable(context.state)),
        );
        let clicked = ui.button(label);
        if clicked && coordinate {
            coordinate_restriction::open(ui);
        } else if clicked && let Some(action) = action {
            context.actions.push(action);
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        unavailable_tooltip(ui, tab, panel, label, display, coordinate);
        if label == coordinate_restriction::LABEL {
            coordinate_restriction::dialog(ui);
        }
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
        if label == "Move Bodies" && crate::move_body_ribbon::is_main(context.owners.customization)
        {
            continue;
        }
        let (position, size) = icons.appearance.cell(index);
        icons.cursor(ui, origin, position);
        if label == "Shell" && !crate::shell_ribbon::is_expanded(context.owners.customization) {
            continue;
        }
        let action = crate::shell_ribbon::additional_action(label);
        let coordinate = coordinate_restriction::ready(label, context.state);
        let disabled = ui.begin_disabled(
            !coordinate
                && action
                    .as_ref()
                    .is_none_or(|action| !action.dispatchable(context.state)),
        );
        let clicked = icons.button(ui, label, glyph, size);
        if clicked && coordinate {
            coordinate_restriction::open(ui);
        } else if clicked && let Some(action) = action {
            context.actions.push(action);
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        unavailable_tooltip(ui, tab, panel, label, display, coordinate);
        if label == coordinate_restriction::LABEL {
            coordinate_restriction::dialog(ui);
        }
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
            ("Shell", Glyph::Shell),
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
    available: bool,
) {
    if !crate::widget_abi::disabled_item_mouse_hovered(ui) {
        return;
    }
    if ui.is_mouse_clicked(imgui::MouseButton::Right) {
        display.request_expansion_command_context(tab, panel, label);
    }
    if label == "Shell" {
        ui.tooltip_text("Shell\nRemove faces to create a hollow solid.");
        return;
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
    if available {
        ui.tooltip_text(format!("{label}\n{help}"));
        return;
    }
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
