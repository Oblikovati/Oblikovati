// SPDX-License-Identifier: GPL-2.0-only
//! Layout of the Customize window and its command lists.
use super::{RibbonCustomization, Ui};

const DEFAULT_SIZE: [f32; 2] = [760., 540.];

pub(super) fn fit_window(ui: &Ui) {
    crate::widget_abi::fit_dialog_with_default_size(ui, "Customize", [340., 185.], DEFAULT_SIZE);
}

pub(super) fn list_size(ui: &Ui, default_width: f32, minimum_width: f32) -> [f32; 2] {
    let [width, height] = ui.window_size();
    [
        (default_width + (width - DEFAULT_SIZE[0]) * 0.5).max(minimum_width),
        (200. + height - DEFAULT_SIZE[1]).max(80.),
    ]
}

impl RibbonCustomization {
    pub(super) fn tabs(&mut self, ui: &Ui) {
        self.ribbon_tab_active = false;
        self.marking_selector_open = false;
        if let Some(_tabs) = ui.tab_bar("##customize-tabs") {
            if let Some(_ribbon) = ui.tab_item("Ribbon") {
                self.ribbon_tab_active = true;
                ui.text("Customize user commands");
                self.lists(ui);
            }
            if let Some(_keyboard) = ui.tab_item("Keyboard") {
                self.keyboard_catalog(ui);
            }
            if let Some(_marking) = ui.tab_item("Marking Menu") {
                self.marking_editor(ui);
            }
        }
    }

    fn lists(&mut self, ui: &Ui) {
        ui.group(|| self.source_list(ui));
        ui.same_line();
        ui.group(|| self.transfer(ui));
        ui.same_line();
        ui.group(|| self.target_list(ui));
        ui.same_line();
        ui.group(|| self.reorder(ui));
    }
}
