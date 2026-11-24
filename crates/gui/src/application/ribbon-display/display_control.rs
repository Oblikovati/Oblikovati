// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl RibbonDisplayState {
    /// Draw the display control and its menu, for example at `[900., 35.]`.
    pub(crate) fn control(
        &mut self,
        ui: &Ui,
        position: [f32; 2],
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
        open_menu: bool,
    ) {
        crate::widget_abi::cursor_position(ui, position);
        let clicked = crate::widget_abi::button(ui, "Ribbon Display", [0.; 2]);
        let activate = self.control_press.button(ui, (), clicked);
        if tips.tabs() && tips.matching("X2") {
            crate::ribbon_key_tips::badge(ui, "X2");
        }
        if activate || self.selector_pointer_press(ui) {
            self.advance();
        }
        if ui.is_item_hovered() {
            ui.tooltip_text(self.control_tooltip());
        }
        ui.same_line();
        self.display_menu_button(ui, tips, open_menu);
    }

    fn display_menu_button(
        &mut self,
        ui: &Ui,
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
        open_menu: bool,
    ) {
        let clicked = crate::widget_abi::button(ui, "v##ribbon-display-menu", [0.; 2]);
        let activate = self.menu_press.button(ui, (), clicked);
        if tips.tabs() && tips.matching("X3") {
            crate::ribbon_key_tips::badge(ui, "X3");
        }
        if activate || open_menu {
            ui.open_popup("##ribbon-display-menu");
        }
        self.keep_temporary_on_click =
            ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
        if let Some(_menu) = ui.begin_popup("##ribbon-display-menu") {
            self.menu(ui);
        } else {
            self.display_menu_cursor = None;
        }
    }
}
