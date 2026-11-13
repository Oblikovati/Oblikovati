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
        muted_command_position(ui, origin, icons, index);
        muted_command(ui, label, reason, icons, None, false);
    }
}
pub(super) fn part_modify_controls(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    hole_tip: bool,
) {
    use crate::icons::IconGlyph;
    muted_icon_commands(
        ui,
        origin,
        icons,
        &[
            ("Hole", Some(IconGlyph::Hole)),
            ("Fillet", Some(IconGlyph::Fillet)),
            ("Chamfer", Some(IconGlyph::Chamfer)),
        ],
        hole_tip,
    );
}
pub(super) fn muted_icon_commands(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    commands: &[(&str, Option<crate::icons::IconGlyph>)],
    hole_tip: bool,
) {
    for (index, (label, glyph)) in commands.iter().enumerate() {
        muted_command_position(ui, origin, icons, index);
        muted_command(
            ui,
            label,
            NOT_PORTED,
            icons,
            *glyph,
            hole_tip && *label == "Hole",
        );
    }
}
fn muted_command_position(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    index: usize,
) {
    let column = (index / 2) as f32 * 116.;
    let row = (index % 2) as f32 * 30.;
    icons.cursor(ui, origin, [column, row]);
}
fn muted_command(
    ui: &Ui,
    label: &str,
    reason: &str,
    icons: &crate::icons::GuiRibbonIcons,
    icon: Option<crate::icons::IconGlyph>,
    hole_tip: bool,
) {
    let disabled = ui.begin_disabled(true);
    match icon {
        Some(glyph) => icons.button(ui, label, glyph, [108., 26.]),
        None => crate::widget_abi::button(ui, label, icons.cell([108., 26.])),
    };
    drop(disabled);
    if hole_tip {
        crate::ribbon_key_tips::badge(ui, "HO");
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(reason);
    }
}
