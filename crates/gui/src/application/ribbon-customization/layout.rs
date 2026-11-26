// SPDX-License-Identifier: GPL-2.0-only
//! Layout of the Customize window and its command lists.
use super::{RibbonCustomization, Ui};
use imgui::StyleVar;

const DEFAULT_SIZE: [f32; 2] = [760., 540.];
pub(super) const TARGET_OPTIONS_X: [f32; 2] = [132., 180.];

pub(super) fn source_sort_header(ui: &Ui, width: f32) -> bool {
    let padding_x = ui.clone_style().frame_padding[0];
    let _padding = ui.push_style_var(StyleVar::FramePadding([padding_x, 0.]));
    let _align = ui.push_style_var(StyleVar::ButtonTextAlign([0., 0.5]));
    ui.button_with_size("Name##available-sort", [width, 0.])
}

pub(super) fn target_headings(ui: &Ui) {
    let origin = ui.cursor_pos()[0] + ui.clone_style().window_padding[0];
    ui.text("Name");
    for (offset, label) in TARGET_OPTIONS_X.into_iter().zip(["Large", "Text"]) {
        let center = origin + offset + ui.frame_height() * 0.5;
        same_row_at(ui, center - ui.calc_text_size(label)[0] * 0.5);
        ui.text(label);
    }
}

pub(super) fn same_row_at(ui: &Ui, x: f32) {
    ui.same_line();
    crate::widget_abi::cursor_position(ui, [x, ui.cursor_pos()[1]]);
}

pub(super) fn fit_window(ui: &Ui) {
    crate::widget_abi::fit_dialog_beside(
        ui,
        "Customize",
        [340., 185.],
        DEFAULT_SIZE,
        (c"Shell", 360.),
    );
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
