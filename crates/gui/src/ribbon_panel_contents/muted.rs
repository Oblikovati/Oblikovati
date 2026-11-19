// SPDX-License-Identifier: GPL-2.0-only
//! Shared layout for unavailable ribbon commands.
use super::NOT_PORTED;
use imgui::Ui;

pub(super) fn muted_commands(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    labels: &[&str],
    reason: &str,
) {
    for (index, label) in labels.iter().enumerate() {
        let size = muted_command_position(ui, origin, icons, index);
        muted_command(ui, label, reason, icons, None, false, size);
    }
}
pub(super) fn manage_insert_commands(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
) {
    muted_commands(
        ui,
        origin,
        icons,
        &[
            "Derive",
            "Feature",
            "Insert iFeature",
            "Insert Object",
            "Import",
        ],
        NOT_PORTED,
    );
}
pub(super) fn part_modify_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &crate::GuiCommandState,
    actions: &mut Vec<crate::GuiShellAction>,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
) {
    let icons = owners.icons;
    use crate::icons::IconGlyph;
    let size = muted_command_position(ui, origin, icons, 0);
    let action = crate::GuiShellAction::OpenHole;
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if icons.button(ui, "Hole", IconGlyph::Hole, size) {
        actions.push(action);
    }
    drop(disabled);
    if owners.hole_tip {
        crate::ribbon_key_tips::badge(ui, "HO");
    }
    let size = muted_command_position(ui, origin, icons, 1);
    let action = crate::GuiShellAction::OpenFillet;
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if icons.button(ui, "Fillet", IconGlyph::Fillet, size) {
        actions.push(action);
    }
    drop(disabled);
    let size = muted_command_position(ui, origin, icons, 2);
    let action = crate::GuiShellAction::OpenChamfer;
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if icons.button(ui, "Chamfer", IconGlyph::Chamfer, size) {
        actions.push(action);
    }
    drop(disabled);
    if !crate::shell_ribbon::is_expanded(owners.customization) {
        let size = muted_command_position(ui, origin, icons, 3);
        let right = owners
            .customization
            .builtin_draw(ui, crate::shell_ribbon::ID, false, || {
                icons.command_button(
                    ui,
                    "Shell",
                    crate::GuiShellAction::OpenShellFeature,
                    state,
                    size,
                    actions,
                );
            });
        owners.quick_access.command_took_right_click |= right;
    }
    if crate::move_body_ribbon::is_main(owners.customization) {
        let size = muted_command_position(ui, origin, icons, 4);
        crate::move_body_ribbon::main_button(
            ui,
            size,
            state,
            actions,
            icons,
            owners.customization,
            owners.quick_access,
        );
    }
}
pub(super) fn muted_icon_commands(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    commands: &[(&str, Option<crate::icons::IconGlyph>)],
    hole_tip: bool,
) {
    for (index, (label, glyph)) in commands.iter().enumerate() {
        let size = muted_command_position(ui, origin, icons, index);
        muted_command(
            ui,
            label,
            NOT_PORTED,
            icons,
            *glyph,
            hole_tip && *label == "Hole",
            size,
        );
    }
}
fn muted_command_position(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    index: usize,
) -> [f32; 2] {
    let (position, size) = icons.appearance.cell(index);
    icons.cursor(ui, origin, position);
    size
}
pub(super) fn muted_command(
    ui: &Ui,
    label: &str,
    reason: &str,
    icons: &crate::icons::GuiRibbonIcons,
    icon: Option<crate::icons::IconGlyph>,
    hole_tip: bool,
    size: [f32; 2],
) {
    let disabled = ui.begin_disabled(true);
    match icon {
        Some(glyph) => icons.button(ui, label, glyph, size),
        None => crate::widget_abi::button(ui, label, icons.cell(size)),
    };
    drop(disabled);
    if hole_tip {
        crate::ribbon_key_tips::badge(ui, "HO");
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(reason);
    }
}
