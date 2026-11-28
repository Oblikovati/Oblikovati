// SPDX-License-Identifier: GPL-2.0-only
//! Transient eligibility tip and failed-edit modals for Part Modify Mark (#3918).
use crate::widget_abi;
use imgui::Ui;

pub(super) const NOTICE_TITLE: &str = "Mark";
pub(super) const MULTI_BODY_MESSAGE: &str =
    "Mark is not supported when multiple solid bodies exist.";
pub(super) const NO_VOLUME_MESSAGE: &str = "Mark failed. The selected component has no volume.";
pub(super) const EDIT_FAILED_TITLE: &str = "Edit Mark Feature Failed";

#[derive(Default)]
pub(super) struct MarkModals {
    notice: Option<&'static str>,
    notice_open: bool,
    edit_failed: Option<String>,
    edit_failed_open: bool,
}

pub(super) enum MarkModalOutcome {
    None,
    CloseMark,
}

impl MarkModals {
    pub(super) fn reset(&mut self) {
        self.notice = None;
        self.notice_open = false;
        self.edit_failed = None;
        self.edit_failed_open = false;
    }

    pub(super) fn show_multi_body(&mut self) {
        self.notice = Some(MULTI_BODY_MESSAGE);
    }

    pub(super) fn show_no_volume(&mut self) {
        self.notice = Some(NO_VOLUME_MESSAGE);
    }

    pub(super) fn show_edit_failed(&mut self, message: String) {
        self.edit_failed = Some(message);
    }

    pub(super) fn any_active(&self) -> bool {
        self.notice.is_some() || self.edit_failed.is_some()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> MarkModalOutcome {
        self.draw_notice(ui);
        if self.draw_edit_failed(ui) {
            return MarkModalOutcome::CloseMark;
        }
        MarkModalOutcome::None
    }

    fn draw_notice(&mut self, ui: &Ui) {
        if self.notice.is_some() && !self.notice_open {
            ui.open_popup(NOTICE_TITLE);
        }
        self.notice_open = self.notice.is_some();
        let Some(_popup) = ui
            .modal_popup_config(NOTICE_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        let Some(message) = self.notice else {
            ui.close_current_popup();
            return;
        };
        ui.text_wrapped(message);
        ui.spacing();
        if widget_abi::button(ui, "OK", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.notice = None;
            self.notice_open = false;
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
