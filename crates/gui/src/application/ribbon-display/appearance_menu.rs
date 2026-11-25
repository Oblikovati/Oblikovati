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
        self.retain_appearance_on_escape = self.context_title == Some(super::RibbonPanel::Create);
        self.capture_submenu_bounds(ui);
        for appearance in RibbonAppearance::ALL {
            if ui
                .menu_item_config(appearance.label())
                .selected(customization.appearance() == appearance)
                .build()
            {
                customization.select_appearance(appearance);
                self.finish_title_context_selection();
            }
        }
        if ui.menu_item("Reset Ribbon") {
            self.request_reset();
            self.close_context_menu = true;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ribbon::RibbonTab;
    use crate::ribbon_panels::RibbonPanel;

    #[test]
    fn appearance_selection_requests_only_its_title_expansion_close() {
        let mut display = RibbonDisplayState::default();
        display.finish_title_context_selection();
        assert!(!display.take_context_expansion_close());
        display.request_expansion_context(RibbonTab::Model, RibbonPanel::PartModify);
        display.finish_title_context_selection();
        assert!(display.take_context_expansion_close());
        assert!(!display.take_context_expansion_close());
        display.request_expansion_command_context(
            RibbonTab::Model,
            RibbonPanel::PartModify,
            "Move Bodies",
        );
        display.finish_title_context_selection();
        assert!(!display.take_context_expansion_close());
    }
}
