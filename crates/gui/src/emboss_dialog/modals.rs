// SPDX-License-Identifier: GPL-2.0-only
//! Transient Sketch Doctor, Edit Failed, and Range Warning modals owned by the dockable Emboss panel.
use crate::widget_abi;
use imgui::Ui;

pub(super) const SKETCH_DOCTOR_TITLE: &str = "Sketch Doctor";
pub(super) const EDIT_FAILED_TITLE: &str = "Edit Emboss Feature Failed";
pub(super) const RANGE_WARNING_TITLE: &str = "Emboss Warning";

#[derive(Default)]
pub(super) struct EmbossModals {
    sketch_doctor: Option<String>,
    sketch_doctor_open: bool,
    edit_failed: Option<String>,
    edit_failed_open: bool,
    range_warning: Option<String>,
    range_warning_open: bool,
}

pub(super) enum ModalOutcome {
    None,
    CloseEmboss,
}

impl EmbossModals {
    pub(super) fn reset(&mut self) {
        self.sketch_doctor = None;
        self.sketch_doctor_open = false;
        self.edit_failed = None;
        self.edit_failed_open = false;
        self.range_warning = None;
        self.range_warning_open = false;
    }

    pub(super) fn show_sketch_doctor(&mut self, message: String) {
        if self.sketch_doctor.as_deref() != Some(message.as_str()) {
            self.sketch_doctor = Some(message);
        }
    }

    pub(super) fn show_edit_failed(&mut self, message: String) {
        self.edit_failed = Some(message);
    }

    pub(super) fn show_range_warning(&mut self, message: String) {
        self.range_warning = Some(message);
    }

    pub(super) fn any_active(&self) -> bool {
        self.sketch_doctor.is_some() || self.edit_failed.is_some() || self.range_warning.is_some()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> ModalOutcome {
        if self.draw_sketch_doctor(ui) {
            return ModalOutcome::CloseEmboss;
        }
        self.draw_edit_failed(ui);
        self.draw_range_warning(ui);
        ModalOutcome::None
    }

    fn draw_sketch_doctor(&mut self, ui: &Ui) -> bool {
        if self.sketch_doctor.is_some() && !self.sketch_doctor_open {
            ui.open_popup(SKETCH_DOCTOR_TITLE);
        }
        self.sketch_doctor_open = self.sketch_doctor.is_some();
        let Some(_popup) = ui
            .modal_popup_config(SKETCH_DOCTOR_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(message) = self.sketch_doctor.clone() else {
            ui.close_current_popup();
            return false;
        };
        ui.text("Diagnostic: Open Loop");
        ui.text_wrapped(&message);
        ui.spacing();
        let cancel = widget_abi::button(ui, "Cancel", [72., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if cancel {
            self.sketch_doctor = None;
            self.sketch_doctor_open = false;
            ui.close_current_popup();
        }
        cancel
    }

    fn draw_edit_failed(&mut self, ui: &Ui) {
        if self.edit_failed.is_some() && !self.edit_failed_open {
            ui.open_popup(EDIT_FAILED_TITLE);
        }
        self.edit_failed_open = self.edit_failed.is_some();
        let Some(_popup) = ui
            .modal_popup_config(EDIT_FAILED_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        let Some(message) = self.edit_failed.clone() else {
            ui.close_current_popup();
            return;
        };
        ui.text("Edit Emboss Feature Failed");
        ui.text_wrapped(&message);
        ui.spacing();
        if widget_abi::button(ui, "Edit", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.edit_failed = None;
            self.edit_failed_open = false;
            ui.close_current_popup();
        }
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
        if !message.contains("out of the expected range") {
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
