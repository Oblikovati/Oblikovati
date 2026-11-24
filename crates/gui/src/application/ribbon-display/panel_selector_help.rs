// SPDX-License-Identifier: GPL-2.0-only
//! Context help for the direct ribbon panel selector.
use imgui::{Key, Ui};

const HELP: &str = "Ribbon Panel Help";
const ARROW: &str = "The down arrow displays more ribbon panels.";
const SHOW: &str = "Select a checkbox to show its panel.";
const HIDE: &str = "Clear a checkbox to hide its panel.";

pub(super) fn draw(ui: &Ui, show_hint: bool) {
    if show_hint && !crate::widget_abi::any_popup_open(ui) {
        if ui.is_key_pressed_no_repeat(Key::F1) {
            ui.open_popup(HELP);
        } else {
            ui.tooltip(|| {
                ui.text("Show Panels");
                ui.separator();
                ui.text(ARROW);
                ui.text(SHOW);
                ui.text(HIDE);
                ui.spacing();
                ui.text("Press F1 for more help.");
            });
        }
    }
    let Some(_popup) = ui
        .modal_popup_config(HELP)
        .always_auto_resize(true)
        .begin_popup()
    else {
        return;
    };
    ui.text(SHOW);
    ui.text(HIDE);
    ui.spacing();
    ui.text("Arrow keys move between panels.");
    ui.text("Space changes the focused checkbox.");
    ui.text("Escape closes the panel list and keeps your choices.");
    ui.spacing();
    if ui.button("Close") || ui.is_key_pressed_no_repeat(Key::Escape) {
        ui.close_current_popup();
    }
}
