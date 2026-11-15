// SPDX-License-Identifier: GPL-2.0-only
//! Appearance choices save the active ribbon layout.
use super::{RibbonDisplayState, submenu_row_pressed};
use crate::ribbon_command_layout::RibbonAppearance;
use imgui::Ui;

const APPEARANCE_MENU: &str = "##ribbon-appearance";
impl RibbonDisplayState {
    pub(super) fn appearance_menu(
        &mut self,
        ui: &Ui,
        submenu_open: bool,
        customization: &mut crate::application::ribbon_customization::RibbonCustomization,
    ) {
        if submenu_row_pressed(ui, "Ribbon Appearance", submenu_open) {
            ui.open_popup(APPEARANCE_MENU);
        }
        let at = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        crate::widget_abi::next_window_position(ui, at);
        let Some(_list) = ui.begin_popup(APPEARANCE_MENU) else {
            return;
        };
        self.capture_submenu_bounds(ui);
        for appearance in RibbonAppearance::ALL {
            if ui
                .menu_item_config(appearance.label())
                .selected(customization.appearance() == appearance)
                .build()
            {
                customization.select_appearance(appearance);
                self.close_context_menu = true;
            }
        }
        if ui.menu_item("Reset Ribbon") {
            self.reset_requested = true;
            self.close_context_menu = true;
        }
    }
}
