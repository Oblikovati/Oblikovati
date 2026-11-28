// SPDX-License-Identifier: GPL-2.0-only
//! Exchange file picker modal launched by the Import command.
use crate::application::document_saving::file_list::FileRequest;
use crate::application::document_saving::folder_browser::{
    FileTypeList, FolderBrowser, center_dialog, file_name_field, file_type_field, full_path,
};
use crate::application::file_system::NativeFolders;
use crate::widget_abi;
use imgui::{Key, Ui};
use std::{path::Path, rc::Rc};

pub(super) const PICKER_TITLE: &str = "Import File";
const PICKER_SIZE: [f32; 2] = [860., 480.];

pub(super) enum ImportPickerOutcome {
    Pending,
    OpenFile(String),
    OpenOptions(String),
    Cancelled,
}

pub(super) struct ImportPicker {
    browser: FolderBrowser,
    open: bool,
    opening: bool,
    file_name: String,
    types: FileTypeList,
    file_type: usize,
    popup_was_open: bool,
}

impl Default for ImportPicker {
    fn default() -> Self {
        let mut browser = FolderBrowser::new(Rc::new(NativeFolders));
        browser.folders_select_first = true;
        Self {
            browser,
            open: false,
            opening: false,
            file_name: String::new(),
            types: FileTypeList::importing(),
            file_type: 0,
            popup_was_open: false,
        }
    }
}

impl ImportPicker {
    pub(super) fn is_open(&self) -> bool {
        self.open
    }

    pub(super) fn is_opening(&self) -> bool {
        self.open && (self.opening || self.browser.selection_redraw)
    }

    pub(super) fn open_in(&mut self, initial_dir: Option<&Path>) {
        if let Some(dir) = initial_dir.filter(|path| path.is_dir()) {
            self.browser.folder = dir.display().to_string();
        }
        self.file_name.clear();
        self.browser.begin_showing();
        self.browser.refresh();
        self.open = true;
        self.opening = true;
        self.popup_was_open = false;
    }

    pub(super) fn reopen_current(&mut self) {
        self.open = true;
        self.opening = true;
        self.popup_was_open = false;
    }

    pub(super) fn close(&mut self) {
        self.open = false;
        self.opening = false;
    }

    fn resolved_selected(&self) -> Option<String> {
        let candidate = self.file_name.trim();
        if candidate.is_empty() {
            return None;
        }
        Some(if Path::new(candidate).is_absolute() {
            candidate.to_string()
        } else {
            full_path(&self.browser.folder, candidate)
        })
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> ImportPickerOutcome {
        if !self.open {
            return ImportPickerOutcome::Pending;
        }
        if std::mem::take(&mut self.opening) {
            ui.open_popup(PICKER_TITLE);
        }
        center_dialog(ui, PICKER_SIZE);
        let Some(_modal) = ui.begin_modal_popup(PICKER_TITLE) else {
            return ImportPickerOutcome::Pending;
        };
        let filter = self.types.filter(self.file_type);
        if let Some(picked) = self.browser.controls(ui, "Look in", filter) {
            self.file_name = picked;
        }
        if let FileRequest::Open(file) = self.browser.take_request() {
            self.file_name = file;
            if let Some(resolved) = self.resolved_selected() {
                ui.close_current_popup();
                self.close();
                return ImportPickerOutcome::OpenFile(resolved);
            }
        }
        ui.separator();
        ui.dummy([0., 4.]);
        let submitted = file_name_field(ui, &mut self.file_name);
        file_type_field(ui, &self.types, &mut self.file_type);

        let selected = self.resolved_selected();
        let can_open = self.browser.highlighted_folder().is_some() || selected.is_some();
        let can_options = selected.as_deref().is_some_and(is_stl_path);

        ui.dummy([0., 4.]);
        let options_clicked = {
            let _disabled = (!can_options).then(|| ui.begin_disabled(true));
            widget_abi::button(ui, "Options", [84., 28.])
        };
        ui.same_line_with_spacing(0., 12.);
        let open_clicked = {
            let _disabled = (!can_open).then(|| ui.begin_disabled(true));
            widget_abi::button(ui, "Open", [84., 28.]) || (submitted && can_open)
        };
        ui.same_line_with_spacing(0., 8.);
        let escape_pressed = ui.is_key_pressed_no_repeat(Key::Escape);
        let cancel_clicked = widget_abi::button(ui, "Cancel", [84., 28.])
            || (escape_pressed && !self.popup_was_open);
        self.popup_was_open = widget_abi::any_popup_open(ui);

        if options_clicked && let Some(path) = selected {
            ui.close_current_popup();
            self.open = false;
            return ImportPickerOutcome::OpenOptions(path);
        }
        if open_clicked && can_open {
            if self.browser.enter_highlighted_folder() {
                return ImportPickerOutcome::Pending;
            }
            if let Some(path) = selected {
                ui.close_current_popup();
                self.close();
                return ImportPickerOutcome::OpenFile(path);
            }
        }
        if cancel_clicked {
            ui.close_current_popup();
            self.close();
            self.file_name.clear();
            return ImportPickerOutcome::Cancelled;
        }
        ImportPickerOutcome::Pending
    }
}

pub(super) fn is_stl_path(path: &str) -> bool {
    Path::new(path)
        .extension()
        .and_then(|ext| ext.to_str())
        .is_some_and(|ext| ext.eq_ignore_ascii_case("stl"))
}
