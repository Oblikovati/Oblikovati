// SPDX-License-Identifier: GPL-2.0-only
//! Transient prerequisite notice and failed-edit modals for Part Modify Thread (#3918).
use crate::widget_abi;
use imgui::Ui;

pub(super) const NO_FACES_TITLE: &str = "Thread";
pub(super) const NO_FACES_MESSAGE: &str = "Could not find cylindrical or conical faces to thread.";
pub(super) const EDIT_FAILED_TITLE: &str = "Edit Thread Feature Failed";

#[derive(Default)]
pub(super) struct ThreadModals {
    no_faces: bool,
    no_faces_open: bool,
    edit_failed: Option<String>,
    edit_failed_open: bool,
}

pub(super) enum ThreadModalOutcome {
    None,
    CloseThread,
}

impl ThreadModals {
    pub(super) fn reset(&mut self) {
        self.no_faces = false;
        self.no_faces_open = false;
        self.edit_failed = None;
        self.edit_failed_open = false;
    }

    pub(super) fn show_no_faces(&mut self) {
        self.no_faces = true;
    }

    pub(super) fn show_edit_failed(&mut self, message: String) {
        self.edit_failed = Some(message);
    }

    pub(super) fn any_active(&self) -> bool {
        self.no_faces || self.edit_failed.is_some()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> ThreadModalOutcome {
        self.draw_no_faces(ui);
        if self.draw_edit_failed(ui) {
            return ThreadModalOutcome::CloseThread;
        }
        ThreadModalOutcome::None
    }

    fn draw_no_faces(&mut self, ui: &Ui) {
        if self.no_faces && !self.no_faces_open {
            ui.open_popup(NO_FACES_TITLE);
        }
        self.no_faces_open = self.no_faces;
        let Some(_popup) = ui
            .modal_popup_config(NO_FACES_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        if !self.no_faces {
            ui.close_current_popup();
            return;
        }
        ui.text_wrapped(NO_FACES_MESSAGE);
        ui.spacing();
        if widget_abi::button(ui, "OK", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.no_faces = false;
            self.no_faces_open = false;
            ui.close_current_popup();
        }
    }

    fn draw_edit_failed(&mut self, ui: &Ui) -> bool {
        if self.edit_failed.is_some() && !self.edit_failed_open {
            ui.open_popup(EDIT_FAILED_TITLE);
        }
        self.edit_failed_open = self.edit_failed.is_some();
        let Some(_popup) = ui
            .modal_popup_config(EDIT_FAILED_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(message) = self.edit_failed.clone() else {
            ui.close_current_popup();
            return false;
        };
        ui.text(EDIT_FAILED_TITLE);
        if !message.is_empty() {
            ui.text_wrapped(&message);
        }
        ui.spacing();
        let edit = widget_abi::button(ui, "Edit", [64., 28.]);
        ui.same_line_with_spacing(0., 8.);
        let cancel = widget_abi::button(ui, "Cancel", [72., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if edit || cancel {
            self.edit_failed = None;
            self.edit_failed_open = false;
            ui.close_current_popup();
        }
        cancel
    }
}
