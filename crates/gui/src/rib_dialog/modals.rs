// SPDX-License-Identifier: GPL-2.0-only
//! Transient sketch-required tip, creation-failed, and range-warning modals for dockable Rib.
use crate::widget_abi;
use imgui::Ui;

pub(super) const SKETCH_TIP_TITLE: &str = "Rib - Sketch Required";
pub(super) const CREATE_FAILED_TITLE: &str = "Create rib feature failed";
pub(super) const RANGE_WARNING_TITLE: &str = "Rib Warning";

#[derive(Default)]
pub(super) struct RibModals {
    sketch_tip: bool,
    sketch_tip_open: bool,
    create_failed: Option<String>,
    create_failed_open: bool,
    range_warning: Option<String>,
    range_warning_open: bool,
}

pub(super) enum RibModalOutcome {
    None,
    CloseRib,
}

impl RibModals {
    pub(super) fn reset(&mut self) {
        self.sketch_tip = false;
        self.sketch_tip_open = false;
        self.create_failed = None;
        self.create_failed_open = false;
        self.range_warning = None;
        self.range_warning_open = false;
    }

    pub(super) fn show_sketch_tip(&mut self) {
        self.sketch_tip = true;
    }

    pub(super) fn show_create_failed(&mut self, message: String) {
        self.create_failed = Some(message);
    }

    pub(super) fn show_range_warning(&mut self, message: String) {
        self.range_warning = Some(message);
    }

    pub(super) fn any_active(&self) -> bool {
        self.sketch_tip || self.create_failed.is_some() || self.range_warning.is_some()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> RibModalOutcome {
        self.draw_sketch_tip(ui);
        if self.draw_create_failed(ui) {
            return RibModalOutcome::CloseRib;
        }
        self.draw_range_warning(ui);
        RibModalOutcome::None
    }

    fn draw_sketch_tip(&mut self, ui: &Ui) {
        if self.sketch_tip && !self.sketch_tip_open {
            ui.open_popup(SKETCH_TIP_TITLE);
        }
        self.sketch_tip_open = self.sketch_tip;
        let Some(_popup) = ui
            .modal_popup_config(SKETCH_TIP_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        if !self.sketch_tip {
            ui.close_current_popup();
            return;
        }
        ui.text_wrapped(
            "No visible, unadaptive 2D sketch is available. Make a 2D sketch visible or create a new 2D sketch to define a rib profile.",
        );
        ui.spacing();
        if widget_abi::button(ui, "OK", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.sketch_tip = false;
            self.sketch_tip_open = false;
            ui.close_current_popup();
        }
    }

    fn draw_create_failed(&mut self, ui: &Ui) -> bool {
        if self.create_failed.is_some() && !self.create_failed_open {
            ui.open_popup(CREATE_FAILED_TITLE);
        }
        self.create_failed_open = self.create_failed.is_some();
        let Some(_popup) = ui
            .modal_popup_config(CREATE_FAILED_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(message) = self.create_failed.clone() else {
            ui.close_current_popup();
            return false;
        };
        ui.text("Create rib feature failed");
        if !message.is_empty() {
            ui.text_wrapped(&message);
        }
        ui.spacing();
        let edit = widget_abi::button(ui, "Edit", [64., 28.]);
        ui.same_line_with_spacing(0., 8.);
        let cancel = widget_abi::button(ui, "Cancel", [72., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if edit || cancel {
            self.create_failed = None;
            self.create_failed_open = false;
            ui.close_current_popup();
        }
        cancel
    }

    fn draw_range_warning(&mut self, ui: &Ui) {
        if self.range_warning.is_some() && !self.range_warning_open {
            ui.open_popup(RANGE_WARNING_TITLE);
        }
        self.range_warning_open = self.range_warning.is_some();
        let Some(_popup) = ui
            .modal_popup_config(RANGE_WARNING_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        let Some(message) = self.range_warning.clone() else {
            ui.close_current_popup();
            return;
        };
        ui.text("The expression is out of the expected range.");
        if !message.is_empty() && !message.contains("out of the expected range") {
            ui.text_wrapped(&message);
        }
        ui.spacing();
        if widget_abi::button(ui, "OK", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.range_warning = None;
            self.range_warning_open = false;
            ui.close_current_popup();
        }
    }
}
