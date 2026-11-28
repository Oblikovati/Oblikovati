// SPDX-License-Identifier: GPL-2.0-only
//! Render additional ribbon commands and their context menus.
use super::*;
#[path = "coordinate_restriction.rs"]
mod coordinate_restriction;
pub(super) fn options_flyout(
    ui: &Ui,
    display: &mut RibbonDisplayState,
    icons: &crate::icons::GuiRibbonIcons,
) -> bool {
    use crate::icons::IconGlyph;
    let reset = icons.flyout_button(ui, "Reset UI Layout", IconGlyph::UserInterface);
    if reset {
        display.request_reset();
    }
    let _disabled = ui.begin_disabled(true);
    icons.flyout_button(ui, "iLogic Configuration", IconGlyph::Preferences);
    reset
}

pub(in super::super) fn tool_options(ui: &Ui, display: &mut RibbonDisplayState) -> bool {
    let left = ui.cursor_pos()[0];
    let reset = ui.button("Reset UI Layout");
    if reset {
        display.request_reset();
    }
    let _disabled = ui.begin_disabled(true);
    crate::widget_abi::cursor_position(ui, [left, ui.cursor_pos()[1]]);
    ui.button("iLogic Configuration");
    reset
}

pub(super) fn draw(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    display: &mut RibbonDisplayState,
    bend_notice: &mut super::bend_notice::BendNotice,
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
    let Some(entries) = additional_icon_entries(panel) else {
        return;
    };
    if context.owners.icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
        additional_icons(context, tab, panel, display, entries, bend_notice);
        return;
    }
    for &(label, glyph) in entries {
        if !additional_visible(label, context.owners.customization) {
            continue;
        }
        let action = additional_action(label);
        let coordinate = coordinate_restriction::ready(label, context.state);
        let bend = label == "Bend Part" && super::bend_notice::BendNotice::available(context.state);
        let disabled = ui.begin_disabled(
            !coordinate
                && !bend
                && action
                    .as_ref()
                    .is_none_or(|action| !action.dispatchable(context.state)),
        );
        let clicked = context.owners.icons.flyout_button(ui, label, glyph)
            || bend && display.context_tab_press(ui);
        if clicked && coordinate {
            coordinate_restriction::open(ui);
        } else if clicked && bend {
            bend_notice.request(context.state);
        } else if clicked && let Some(action) = action.as_ref() {
            context.actions.push(action.clone());
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        let available = coordinate
            || bend
            || action
                .as_ref()
                .is_some_and(|action| action.dispatchable(context.state));
        unavailable_tooltip(ui, tab, panel, label, display, available);
        if label == coordinate_restriction::LABEL {
            coordinate_restriction::dialog(ui);
        }
    }
}

pub(super) fn normal_expansion_height(
    ui: &Ui,
    panel: RibbonPanel,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> Option<f32> {
    let rows = if matches!(panel, RibbonPanel::ToolOptions | RibbonPanel::Create) {
        2
    } else {
        additional_icon_entries(panel)?
            .iter()
            .filter(|(label, _)| additional_visible(label, customization))
            .count()
    };
    let style = ui.clone_style();
    let footer = ui.text_line_height() + 2. * style.window_padding[1] + 7.;
    Some(rows as f32 * (crate::icons::FLYOUT_ROW_HEIGHT + style.item_spacing[1]) + footer)
}

fn additional_visible(
    label: &str,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> bool {
    match label {
        "Move Bodies" => !crate::move_body_ribbon::is_main(customization),
        "Shell" => crate::shell_ribbon::is_expanded(customization),
        _ => true,
    }
}
fn additional_icons(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    display: &mut RibbonDisplayState,
    entries: &'static [(&'static str, crate::icons::IconGlyph)],
    bend_notice: &mut super::bend_notice::BendNotice,
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
        let action = additional_action(label);
        let coordinate = coordinate_restriction::ready(label, context.state);
        let bend = label == "Bend Part" && super::bend_notice::BendNotice::available(context.state);
        let disabled = ui.begin_disabled(
            !coordinate
                && !bend
                && action
                    .as_ref()
                    .is_none_or(|action| !action.dispatchable(context.state)),
        );
        let clicked = icons.button(ui, label, glyph, size) || bend && display.context_tab_press(ui);
        if clicked && coordinate {
            coordinate_restriction::open(ui);
        } else if clicked && bend {
            bend_notice.request(context.state);
        } else if clicked && let Some(action) = action.as_ref() {
            context.actions.push(action.clone());
        }
        drop(disabled);
        if label == "Move Bodies" && context.owners.key_tips_active {
            crate::ribbon_key_tips::badge(ui, "OD");
        }
        let available = coordinate
            || bend
            || action
                .as_ref()
                .is_some_and(|action| action.dispatchable(context.state));
        unavailable_tooltip(ui, tab, panel, label, display, available);
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
        RibbonPanel::PartCreate => &[
            ("Coil", Glyph::Coil),
            ("Emboss", Glyph::Emboss),
            ("Rib", Glyph::Rib),
            ("Decal", Glyph::Decal),
            ("Unwrap", Glyph::Unwrap),
        ],
        RibbonPanel::PartModify => &[
            ("Move Bodies", Glyph::MoveBodies),
            ("Bend Part", Glyph::BendPart),
            ("Copy Object", Glyph::CopyObject),
            ("Shell", Glyph::Shell),
            ("Slot", Glyph::Hole),
            ("Combine", Glyph::ExtrudeJoin),
            ("Draft", Glyph::Chamfer),
            ("Thread", Glyph::Thread),
            ("Mark", Glyph::Mark),
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
    use crate::sketch::create_block::{CreateBlockButton, ribbon_button};
    let normal = icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal;
    let button = if normal {
        CreateBlockButton::Flyout
    } else {
        CreateBlockButton::Ribbon(icons.appearance.cell(0).1)
    };
    let origin = ui.cursor_pos();
    ribbon_button(ui, state, create_block, actions, button, icons);
    let disabled = ui.begin_disabled(state.precise_target.is_none() || state.transaction_open);
    if normal {
        precise_input.toggle_flyout(ui, icons);
    } else {
        icons.cursor(ui, origin, icons.appearance.cell(1).0);
        precise_input.toggle_icon(ui, icons);
        icons.cursor(ui, origin, [0., icons.appearance.command_height()]);
    }
    drop(disabled);
}

fn additional_action(label: &str) -> Option<GuiShellAction> {
    match label {
        "Coil" => Some(GuiShellAction::Coil(crate::GuiCoilAction::Open)),
        "Emboss" => Some(GuiShellAction::Emboss(crate::GuiEmbossAction::Open)),
        "Rib" => Some(GuiShellAction::Rib(crate::GuiRibAction::Open)),
        "Decal" => Some(GuiShellAction::Decal(crate::GuiDecalAction::Open)),
        "Unwrap" => Some(GuiShellAction::Unwrap(crate::GuiUnwrapAction::Open)),
        "Move Bodies" => Some(GuiShellAction::OpenMoveBody),
        crate::shell_ribbon::LABEL => Some(GuiShellAction::OpenShellFeature),
        "Slot" => Some(GuiShellAction::Slot(crate::GuiSlotAction::Open)),
        "Combine" => Some(GuiShellAction::Combine(crate::GuiCombineAction::Open)),
        "Draft" => Some(GuiShellAction::Draft(crate::GuiDraftAction::Open)),
        "Thread" => Some(GuiShellAction::Thread(crate::GuiThreadAction::Open)),
        "Mark" => Some(GuiShellAction::Mark(crate::GuiMarkAction::Open)),
        "Constraint Inference Scope" => Some(GuiShellAction::OpenInferenceScope),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn expanded_commands_keep_their_real_actions() {
        assert_eq!(
            additional_action("Shell"),
            Some(GuiShellAction::OpenShellFeature)
        );
        assert_eq!(
            additional_action("Move Bodies"),
            Some(GuiShellAction::OpenMoveBody)
        );
        assert_eq!(
            additional_action("Constraint Inference Scope"),
            Some(GuiShellAction::OpenInferenceScope)
        );
        assert_eq!(additional_action("Bend Part"), None);
    }
}
