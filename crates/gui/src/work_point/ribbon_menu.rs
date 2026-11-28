// SPDX-License-Identifier: GPL-2.0-only
//! 3D Model > Work Features > Point split button and constructor dropdown menu (#3918).
use crate::{
    GuiCommandState, GuiShellAction,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
    widget_abi::{self, IconButtonPresentation},
    work_point::{GuiWorkPointAction, GuiWorkPointMode},
};
use imgui::{StyleColor, Ui};

const POPUP_ID: &str = "##point-variant-menu";

const PRIMARY_CONSTRUCTORS: [GuiWorkPointMode; 4] = [
    GuiWorkPointMode::OnVertex,
    GuiWorkPointMode::ThreePlanes,
    GuiWorkPointMode::TwoLines,
    GuiWorkPointMode::PlaneAndLine,
];

const CENTER_CONSTRUCTORS: [GuiWorkPointMode; 3] = [
    GuiWorkPointMode::LoopCenter,
    GuiWorkPointMode::TorusCenter,
    GuiWorkPointMode::SphereCenter,
];

/// Draw the Point split button and its constructor dropdown inside its ribbon cell.
pub(crate) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    size: [f32; 2],
    actions: &mut Vec<GuiShellAction>,
) {
    let action = GuiShellAction::WorkPoint(GuiWorkPointAction::Start(GuiWorkPointMode::General));
    crate::chrome::command_control(ui, "Point", action, state, actions, || {
        icons.button(
            ui,
            "Point",
            IconGlyph::WorkPoint,
            crate::ribbon_split::primary_size(size),
        )
    });
    if crate::ribbon_split::trigger(ui, icons, "##point-variants", size[1]) {
        ui.open_popup(POPUP_ID);
    }
    draw_variants_popup(ui, state, icons, actions);
}

fn draw_variants_popup(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_popup) = ui.begin_popup(POPUP_ID) else {
        return;
    };
    let _button_color =
        ui.push_style_color(StyleColor::Button, ui.style_color(StyleColor::PopupBg));
    menu_item(ui, state, icons, GuiWorkPointMode::General, actions);
    menu_item(ui, state, icons, GuiWorkPointMode::Grounded, actions);
    ui.separator();
    for mode in PRIMARY_CONSTRUCTORS {
        menu_item(ui, state, icons, mode, actions);
    }
    ui.separator();
    for mode in CENTER_CONSTRUCTORS {
        menu_item(ui, state, icons, mode, actions);
    }
}

fn menu_item(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    mode: GuiWorkPointMode,
    actions: &mut Vec<GuiShellAction>,
) {
    let action = GuiShellAction::WorkPoint(GuiWorkPointAction::Start(mode));
    let _disabled = ui.begin_disabled(!action.dispatchable(state));
    let clicked = widget_abi::icon_button(
        ui,
        mode.label(),
        IconButtonPresentation {
            texture: icons.key(IconGlyph::WorkPoint, IconSize::Small),
            size: icons.cell([265., 22.]),
            icon: icons.logical[IconSize::Small.index()],
            large: false,
            show_caption: true,
        },
    );
    if clicked {
        actions.push(action);
        ui.close_current_popup();
    }
}
