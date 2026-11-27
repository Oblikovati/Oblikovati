// SPDX-License-Identifier: GPL-2.0-only
//! Component file picker modal launched by the Derive command before opening Derived Part.
use crate::application::document_saving::file_list::FileRequest;
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field, file_type_field, full_path,
};
use crate::application::file_system::NativeFolders;
use imgui::Ui;
use std::{path::Path, rc::Rc};

const PICKER_TITLE: &str = "Select Component";
const PICKER_SIZE: [f32; 2] = [860., 480.];

pub(super) enum PickerOutcome {
    Pending,
    Selected(String),
    Cancelled,
}

pub(super) struct ComponentPicker {
    browser: FolderBrowser,
    open: bool,
    opening: bool,
    file_name: String,
    types: FileTypeList,
    file_type: usize,
    escape: DialogEscape,
}

impl Default for ComponentPicker {
    fn default() -> Self {
        let mut browser = FolderBrowser::new(Rc::new(NativeFolders));
        browser.folders_select_first = true;
        Self {
            browser,
            open: false,
            opening: false,
            file_name: String::new(),
            types: FileTypeList::opening(),
            file_type: 0,
            escape: DialogEscape::default(),
        }
    }
}

impl ComponentPicker {
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
        self.escape = DialogEscape::default();
    }

    pub(super) fn close(&mut self) {
        self.open = false;
        self.opening = false;
        self.file_name.clear();
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> PickerOutcome {
        if !self.open {
            return PickerOutcome::Pending;
        }
        if std::mem::take(&mut self.opening) {
            ui.open_popup(PICKER_TITLE);
        }
        center_dialog(ui, PICKER_SIZE);
        let Some(_modal) = ui.begin_modal_popup(PICKER_TITLE) else {
            return PickerOutcome::Pending;
        };
        let filter = self.types.filter(self.file_type);
        if let Some(picked) = self.browser.controls(ui, "Look in", filter) {
            self.file_name = picked;
        }
        if let FileRequest::Open(file) = self.browser.take_request() {
            self.file_name = file;
            let resolved = full_path(&self.browser.folder, &self.file_name);
            ui.close_current_popup();
            self.close();
            return PickerOutcome::Selected(resolved);
        }
        ui.separator();
        ui.dummy([0., 4.]);
        let submitted = file_name_field(ui, &mut self.file_name);
        file_type_field(ui, &self.types, &mut self.file_type);
        let can_open =
            self.browser.highlighted_folder().is_some() || !self.file_name.trim().is_empty();
        let end = dialog_buttons(ui, "Open", can_open, submitted, &mut self.escape);
        match end {
            Some(DialogEnd::Accepted) if can_open => {
                if self.browser.enter_highlighted_folder() {
                    return PickerOutcome::Pending;
                }
                let candidate = self.file_name.trim();
                let resolved = if Path::new(candidate).is_absolute() {
                    candidate.to_string()
                } else {
                    full_path(&self.browser.folder, candidate)
                };
                ui.close_current_popup();
                self.close();
                PickerOutcome::Selected(resolved)
            }
            Some(DialogEnd::Cancelled) => {
                ui.close_current_popup();
                self.close();
                PickerOutcome::Cancelled
            }
            _ => PickerOutcome::Pending,
        }
    }
}
