// SPDX-License-Identifier: GPL-2.0-only
//! Move Bodies uses the same action in both ribbon locations and the toolbar (#4098).
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
const MAIN: &str = "part.model.modify.move-bodies-main";
pub(crate) const LABEL: &str = "Move Bodies";

pub(crate) fn is_main(
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> bool {
    customization.builtin_expanded(MAIN)
}
pub(crate) fn set_main(
    customization: &mut crate::application::ribbon_customization::RibbonCustomization,
    main: bool,
) -> Result<(), String> {
    customization.relocate_builtin(MAIN, main)
}
pub(crate) fn main_button(
    ui: &Ui,
    size: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    icons: &crate::icons::GuiRibbonIcons,
    customization: &mut crate::application::ribbon_customization::RibbonCustomization,
    quick_access: &mut crate::application::quick_access_toolbar::QuickAccessToolbar,
) {
    let action = GuiShellAction::OpenMoveBody;
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if icons.button(ui, LABEL, crate::icons::IconGlyph::MoveBodies, size) {
        actions.push(action);
    }
    drop(disabled);
    let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
    if hovered && ui.is_mouse_clicked(imgui::MouseButton::Right) {
        quick_access.command_took_right_click = true;
    }
    let opened = hovered && ui.is_mouse_released(imgui::MouseButton::Right);
    if opened {
        ui.open_popup("##move-body-main-context");
    }
    let Some(_menu) = ui.begin_popup("##move-body-main-context") else {
        return;
    };
    if opened {
        ui.set_keyboard_focus_here();
    }
    if menu_item(ui, "Move to Expanded Panel", true) {
        customization.relocate_builtin_reported(MAIN, false);
    }
    if menu_item(
        ui,
        "Add to Quick Access Toolbar",
        !quick_access.has_ribbon_command(state, LABEL),
    ) {
        quick_access.add_ribbon_command(state, LABEL);
    }
}

/// Activate a context-menu row with pointer or focused Enter; example: `menu_item(ui, "Move", true)`.
pub(crate) fn menu_item(ui: &Ui, label: &str, enabled: bool) -> bool {
    let clicked = ui.menu_item_config(label).enabled(enabled).build();
    let keyboard = enabled
        && ui.is_item_focused()
        && (ui.is_key_pressed_no_repeat(imgui::Key::Enter)
            || ui.is_key_pressed_no_repeat(imgui::Key::KeypadEnter));
    if keyboard {
        ui.close_current_popup();
    }
    clicked || keyboard
}
