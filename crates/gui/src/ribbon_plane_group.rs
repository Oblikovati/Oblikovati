// SPDX-License-Identifier: GPL-2.0-only
//! Work Plane's offset action and reference constructor group (#4105).
use crate::{
    GuiCommandState, GuiShellAction,
    icons::{GuiRibbonIcons, IconGlyph},
};
use imgui::Ui;

const VARIANTS: [&str; 11] = [
    "Midplane",
    "Three Points",
    "Parallel through Point",
    "Through Axis and Point",
    "Through Two Axes",
    "At Angle to Plane",
    "Tangent to Face",
    "Tangent through Point",
    "Tangent through Axis",
    "Normal to Axis",
    "Torus Midplane",
];
pub(super) const COMMAND_COUNT: usize = VARIANTS.len() + 1;

pub(super) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    ungrouped: bool,
    expanded: bool,
) {
    let origin = ui.cursor_pos();
    let (_, size) = icons.appearance.cell(0);
    crate::chrome::command_control(
        ui,
        "Plane",
        GuiShellAction::OpenOffsetPlane,
        state,
        actions,
        || icons.button(ui, "Plane", IconGlyph::WorkPlane, size),
    );
    if ungrouped {
        draw_individual_variants(ui, icons, origin, expanded);
    } else {
        draw_variants_popup(ui);
    }
}
fn draw_variants_popup(ui: &Ui) {
    ui.same_line();
    if ui.small_button("v##plane-variants") {
        ui.open_popup("##plane-variant-menu");
    }
    let Some(_popup) = ui.begin_popup("##plane-variant-menu") else {
        return;
    };
    let disabled = ui.begin_disabled(true);
    for name in VARIANTS {
        ui.menu_item(name);
        unavailable_reason(ui);
    }
    drop(disabled);
}
fn draw_individual_variants(ui: &Ui, icons: &GuiRibbonIcons, origin: [f32; 2], expanded: bool) {
    for (index, name) in VARIANTS.iter().enumerate() {
        let (position, size) = if expanded {
            ([0., (index as f32 + 1.) * 30.], [170., 26.])
        } else {
            icons.appearance.cell(index + 1)
        };
        icons.cursor(ui, origin, position);
        let disabled = ui.begin_disabled(true);
        icons.button(ui, name, IconGlyph::WorkPlane, size);
        drop(disabled);
        unavailable_reason(ui);
    }
}
fn unavailable_reason(ui: &Ui) {
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("This plane constructor is not available in this build.");
    }
}
