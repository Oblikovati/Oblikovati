// SPDX-License-Identifier: GPL-2.0-only
//! Work Axis split button and constructor dropdown menu in 3D Model > Work Features (#3918).
use crate::{
    GuiCommandState, GuiShellAction, GuiWorkAxisAction, GuiWorkAxisMode,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
    widget_abi::{self, IconButtonPresentation},
};
use imgui::{StyleColor, Ui};

const DROPDOWN_MODES: [GuiWorkAxisMode; 7] = [
    GuiWorkAxisMode::OnLineOrEdge,
    GuiWorkAxisMode::ParallelToLineThroughPoint,
    GuiWorkAxisMode::ThroughTwoPoints,
    GuiWorkAxisMode::IntersectionOfTwoPlanes,
    GuiWorkAxisMode::NormalToPlaneThroughPoint,
    GuiWorkAxisMode::ThroughCircularEdge,
    GuiWorkAxisMode::ThroughRevolvedFace,
];

/// Draw the Axis split command button and its constructor popup menu.
/// Example: `ribbon_axis_group::draw(ui, state, icons, actions, size)`.
pub(super) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    size: [f32; 2],
) {
    let primary = crate::ribbon_split::primary_size(size);
    let action = GuiShellAction::WorkAxis(GuiWorkAxisAction::Start(GuiWorkAxisMode::General));
    crate::chrome::command_control(ui, "Axis", action, state, actions, || {
        icons.button(ui, "Axis", IconGlyph::WorkAxis, primary)
    });
    if crate::ribbon_split::trigger(ui, icons, "##axis-variants", size[1]) {
        ui.open_popup("##axis-variant-menu");
    }
    draw_variants_popup(ui, state, icons, actions);
}

fn draw_variants_popup(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_popup) = ui.begin_popup("##axis-variant-menu") else {
        return;
    };
    let _button_color =
        ui.push_style_color(StyleColor::Button, ui.style_color(StyleColor::PopupBg));
    draw_mode_row(ui, state, icons, actions, GuiWorkAxisMode::General);
    ui.separator();
    for mode in DROPDOWN_MODES {
        draw_mode_row(ui, state, icons, actions, mode);
    }
}

fn draw_mode_row(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    mode: GuiWorkAxisMode,
) {
    let action = GuiShellAction::WorkAxis(GuiWorkAxisAction::Start(mode));
    let _disabled = ui.begin_disabled(!action.dispatchable(state));
    if popup_choice(ui, icons, mode.label()) {
        actions.push(action);
        ui.close_current_popup();
    }
}

fn popup_choice(ui: &Ui, icons: &GuiRibbonIcons, name: &str) -> bool {
    widget_abi::icon_button(
        ui,
        name,
        IconButtonPresentation {
            texture: icons.key(IconGlyph::WorkAxis, IconSize::Small),
            size: icons.cell([320., 22.]),
            icon: icons.logical[IconSize::Small.index()],
            large: false,
            show_caption: true,
        },
    )
}
