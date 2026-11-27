// SPDX-License-Identifier: GPL-2.0-only
//! Browse for a native project and keep the picker open until the owner accepts it.
use crate::application::document_saving::file_list::FileRequest;
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileFilter, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field_focused, full_path,
};
use crate::application::file_system::NativeFolders;
use crate::{GuiCommandState, GuiProjectAction, GuiShellAction};
use imgui::Ui;
use std::{path::Path, rc::Rc};

const TITLE: &str = "Browse for Project";
const SIZE: [f32; 2] = [760., 430.];

#[derive(Default)]
struct ProjectFileDraft {
    opening: bool,
    name: String,
    request: Option<u64>,
    error: Option<String>,
    escape: DialogEscape,
}

pub(super) struct ProjectPicker {
    browser: FolderBrowser,
    draft: Option<ProjectFileDraft>,
}

impl Default for ProjectPicker {
    fn default() -> Self {
        let mut browser = FolderBrowser::new(Rc::new(NativeFolders));
        browser.show_file_properties = false;
        Self {
            browser,
            draft: None,
        }
    }
}

impl ProjectPicker {
    pub(super) fn request(&mut self, location: Option<&str>) {
        self.browser.reset();
        if let Some(parent) = location.and_then(|file| Path::new(file).parent()) {
            self.browser.folder = parent.display().to_string();
            self.browser.refresh();
        }
        self.browser.begin_showing();
        self.draft = Some(ProjectFileDraft {
            opening: true,
            ..Default::default()
        });
    }

    pub(super) fn cancel(&mut self) {
        self.draft = None;
    }

    pub(super) fn needs_frame(&self) -> bool {
        self.draft
            .as_ref()
            .is_some_and(|draft| draft.opening || self.browser.selection_redraw)
    }

    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<String> {
        let draft = self.draft.as_mut()?;
        if std::mem::take(&mut draft.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, SIZE);
        let _popup = ui.begin_modal_popup(TITLE)?;
        if let Some(location) = draft.complete(state) {
            self.cancel();
            ui.close_current_popup();
            return Some(location);
        }
        if draft.controls(ui, &mut self.browser, actions) {
            self.cancel();
            ui.close_current_popup();
        }
        None
    }
}

impl ProjectFileDraft {
    fn complete(&mut self, state: &GuiCommandState) -> Option<String> {
        let (id, result) = state.project_file_result.as_ref()?;
        if self.request != Some(*id) {
            return None;
        }
        self.request = None;
        match result {
            Ok(location) => Some(location.clone()),
            Err(error) => {
                self.error = Some(error.clone());
                None
            }
        }
    }

    fn controls(
        &mut self,
        ui: &Ui,
        browser: &mut FolderBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let project_extension = &oblikovati_api::types::PROJECT_FILE_EXTENSION[1..];
        if let Some(name) =
            browser.controls(ui, "Look in", FileFilter::Extension(project_extension))
        {
            self.name = name;
            self.error = None;
        }
        ui.separator();
        let entered = file_name_field_focused(ui, &mut self.name, false);
        ui.text(format!("Project Files (*.{project_extension})"));
        if let Some(error) = &self.error {
            ui.text_wrapped(error);
        }
        let open_row = if let FileRequest::Open(name) = browser.take_request() {
            self.name = name;
            true
        } else {
            false
        };
        let can_open = self.request.is_none() && !self.name.trim().is_empty();
        match dialog_buttons(ui, "Open", can_open, entered || open_row, &mut self.escape) {
            Some(DialogEnd::Accepted) => self.submit(&browser.folder, actions),
            Some(DialogEnd::Cancelled) => return true,
            None => {}
        }
        false
    }

    fn submit(&mut self, directory: &str, actions: &mut Vec<GuiShellAction>) {
        let request = super::next_project_request();
        self.request = Some(request);
        self.error = None;
        actions.push(GuiShellAction::ProjectFile(GuiProjectAction::Open {
            request,
            path: full_path(directory, self.name.trim()).into(),
        }));
    }
}
