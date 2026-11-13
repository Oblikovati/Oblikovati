// SPDX-License-Identifier: GPL-2.0-only
//! Appearance commands are disclosed without suggesting unfinished layouts work.
use super::{RibbonDisplayState, muted_context_item};
use imgui::Ui;

const APPEARANCE_MENU: &str = "##ribbon-appearance";
impl RibbonDisplayState {
    pub(super) fn appearance_menu(&mut self, ui: &Ui) {
        if crate::widget_abi::dialog_selectable(ui, "Ribbon Appearance", false) {
            ui.open_popup(APPEARANCE_MENU);
        }
        let at = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        crate::widget_abi::next_window_position(ui, at);
        let Some(_list) = ui.begin_popup(APPEARANCE_MENU) else {
            return;
        };
        self.capture_submenu_bounds(ui);
        if ui.menu_item_config("Normal").selected(true).build() {
            self.close_context_menu = true;
        }
        for label in ["Text Off", "Small", "Compact", "Large"] {
            muted_context_item(ui, label);
        }
        if ui.menu_item("Reset Ribbon") {
            self.reset_requested = true;
            self.close_context_menu = true;
        }
    }
}
