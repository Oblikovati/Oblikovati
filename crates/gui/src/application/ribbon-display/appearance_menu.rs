// SPDX-License-Identifier: GPL-2.0-only
//! Appearance commands are disclosed without suggesting unfinished layouts work.
use super::{RibbonDisplayState, muted_context_item, submenu_row_pressed};
use imgui::Ui;

const APPEARANCE_MENU: &str = "##ribbon-appearance";
impl RibbonDisplayState {
    pub(super) fn appearance_menu(&mut self, ui: &Ui, submenu_open: bool) {
        if submenu_row_pressed(ui, "Ribbon Appearance", submenu_open) {
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
        for appearance in crate::ribbon_command_layout::RibbonAppearance::ALL
            .into_iter()
            .skip(1)
        {
            muted_context_item(ui, appearance.label());
        }
        if ui.menu_item("Reset Ribbon") {
            self.reset_requested = true;
            self.close_context_menu = true;
        }
    }
}
