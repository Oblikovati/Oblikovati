// SPDX-License-Identifier: GPL-2.0-only
//! Work Plane's offset action and reference constructor group (#4105).
use crate::{
    GuiCommandState, GuiShellAction,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
    widget_abi::{self, IconButtonPresentation},
};
use imgui::{StyleColor, Ui};

const REFERENCE_VARIANTS: [&str; 5] = [
    "Midplane",
    "Three Points",
    "Parallel through Point",
    "Through Axis and Point",
    "Through Two Axes",
];
const ORIENTATION_VARIANTS: [&str; 6] = [
    "At Angle to Plane",
    "Tangent to Face",
    "Tangent through Point",
    "Tangent through Axis",
    "Normal to Axis",
    "Torus Midplane",
];
pub(super) const COMMAND_COUNT: usize = REFERENCE_VARIANTS.len() + ORIENTATION_VARIANTS.len() + 1;

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
        draw_variants_popup(ui, state, icons, actions);
    }
}
fn draw_variants_popup(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.same_line();
    if ui.small_button("v##plane-variants") {
        ui.open_popup("##plane-variant-menu");
    }
    let Some(_popup) = ui.begin_popup("##plane-variant-menu") else {
        return;
    };
    ui.text("Plane");
    let _button_color =
        ui.push_style_color(StyleColor::Button, ui.style_color(StyleColor::PopupBg));
    let action = GuiShellAction::OpenOffsetPlane;
    let offset_enabled = ui.begin_disabled(!action.dispatchable(state));
    if popup_choice(ui, icons, "Offset from Plane") {
        actions.push(action);
        ui.close_current_popup();
    }
    drop(offset_enabled);
    let disabled = ui.begin_disabled(true);
    for group in [
        REFERENCE_VARIANTS.as_slice(),
        ORIENTATION_VARIANTS.as_slice(),
    ] {
        ui.separator();
        for &name in group {
            popup_choice(ui, icons, name);
            unavailable_reason(ui);
        }
    }
    drop(disabled);
}
fn popup_choice(ui: &Ui, icons: &GuiRibbonIcons, name: &str) -> bool {
    widget_abi::icon_button(
        ui,
        name,
        IconButtonPresentation {
            texture: icons.key(IconGlyph::WorkPlane, IconSize::Small),
            size: icons.cell([215., 22.]),
            icon: icons.logical[IconSize::Small.index()],
            large: false,
            show_caption: true,
        },
    )
}
fn draw_individual_variants(ui: &Ui, icons: &GuiRibbonIcons, origin: [f32; 2], expanded: bool) {
    for (index, name) in REFERENCE_VARIANTS
        .into_iter()
        .chain(ORIENTATION_VARIANTS)
        .enumerate()
    {
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
