// SPDX-License-Identifier: GPL-2.0-only
//! Defer keyboard expansion dismissal until the document owner resolves a model click (#4099).
use super::*;

pub(super) fn keyboard_model_click(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &RibbonDrawOwners<'_>,
    band: f32,
) -> bool {
    if !owners.ribbon.tips.model_commands()
        || !owners.ribbon.panels.has_additional()
        || request.commands.authoring != crate::GuiAuthoringContext::Part
        || !ui.is_mouse_clicked(imgui::MouseButton::Left)
        || ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
    {
        return false;
    }
    let area = owners.workspace.document_area(ui.io().display_size, band);
    let [x, y] = ui.io().mouse_pos;
    (area.left..area.right).contains(&x) && (area.top..area.bottom).contains(&y)
}

pub(super) fn process_ribbon_key_tips(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    frame: &RibbonFrame,
    blocked: bool,
    grouped: bool,
    height: f32,
) -> Option<crate::ribbon_key_tips::KeyTipAction> {
    if !owners.ribbon.panels.has_additional() {
        owners.ribbon.panels.forget_retained_face();
    }
    if keyboard_model_click(ui, request, owners, height) {
        owners.ribbon.panels.begin_model_pick();
        owners.ribbon.tips.defer_pointer_once();
    }
    let cancel_face = !crate::widget_abi::any_popup_open(ui)
        && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        && owners.ribbon.panels.cancel_retained_face();
    if cancel_face {
        owners.ribbon.tips.dismiss();
    }
    let action = owners
        .ribbon
        .tips
        .process(&request.events, &frame.tabs, blocked, grouped);
    if cancel_face {
        owners.ribbon.tips.consumed = true;
    }
    action
}
