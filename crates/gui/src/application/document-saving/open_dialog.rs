// SPDX-License-Identifier: GPL-2.0-only
use super::folder_browser::{
    BROWSER_COLUMN_WIDTH, DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog,
    dialog_buttons, file_name_field, file_type_field, full_path,
};
use crate::application::file_system::{GuiFolders, NativeFolders};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::OpenDocumentArgs;
use std::rc::Rc;

const TITLE: &str = "Open";
const NO_PREVIEW: &str = "No preview image";
/// The list's last row: Open starts by showing every file in the folder.
const ALL_DOCUMENT_TYPES: usize = 4;

struct OpenDraft {
    name: String,
    types: FileTypeList,
    file_type: usize,
    escape: DialogEscape,
    opening: bool,
}
pub(crate) struct OpenDialog {
    browser: FolderBrowser,
    draft: Option<OpenDraft>,
}
impl Default for OpenDialog {
    fn default() -> Self {
        Self::with_folders(Rc::new(NativeFolders))
    }
}
impl OpenDialog {
    pub(crate) fn with_folders(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            browser: FolderBrowser::new(folders),
            draft: None,
        }
    }
    /// Open at the last folder; example: `dialog.request()` from the File menu.
    pub(crate) fn request(&mut self) {
        self.browser.refresh();
        self.draft = Some(OpenDraft {
            name: String::new(),
            types: FileTypeList::opening(),
            file_type: ALL_DOCUMENT_TYPES,
            escape: DialogEscape::default(),
            opening: true,
        });
    }
    /// A frame must follow so the popup opens without waiting for an event.
    pub(crate) fn is_opening(&self) -> bool {
        self.draft.as_ref().is_some_and(|draft| draft.opening)
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        let name = self.draft.as_ref().map_or("", |draft| draft.name.as_str());
        if name.contains('\0') || self.browser.folder.contains('\0') {
            return Err(format!(
                "open dialog name {name:?}, folder {:?}: expected text without NUL",
                self.browser.folder
            ));
        }
        Ok(())
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        pictures: &crate::application::document_tabs::previews::TabPreviews,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(draft) = &mut self.draft else {
            return;
        };
        if std::mem::take(&mut draft.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, [1000., 420.]);
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        if contents(ui, state, pictures, &mut self.browser, draft, actions).is_some() {
            self.draft = None;
            ui.close_current_popup();
        }
    }
}
fn contents(
    ui: &Ui,
    state: &GuiCommandState,
    pictures: &crate::application::document_tabs::previews::TabPreviews,
    browser: &mut FolderBrowser,
    draft: &mut OpenDraft,
    actions: &mut Vec<GuiShellAction>,
) -> Option<DialogEnd> {
    ui.columns(2, "open-layout", true);
    ui.set_column_width(0, BROWSER_COLUMN_WIDTH);
    if let Some(name) = browser.controls(ui, "Look in", draft.types.filter(draft.file_type)) {
        draft.name = name;
    }
    ui.next_column();
    ui.child_window("##open-preview")
        .size([0., 110.])
        .border(true)
        .build(|| {
            let path = full_path(&browser.folder, &draft.name);
            match pictures.picture_of_file(&path) {
                Some(image) => super::save_as::draw_preview_image(ui, image),
                None => ui.text_disabled(NO_PREVIEW),
            }
        });
    ui.columns(1, "open-layout-end", false);
    ui.separator();
    let entered = file_name_field(ui, &mut draft.name);
    file_type_field(ui, &draft.types, &mut draft.file_type);
    let action = open_action(&full_path(&browser.folder, &draft.name));
    let can_open = !draft.name.is_empty() && action.dispatchable(state);
    let end = dialog_buttons(ui, "Open", can_open, entered, &mut draft.escape);
    if end == Some(DialogEnd::Accepted) {
        actions.push(action);
    }
    end
}
fn open_action(path: &str) -> GuiShellAction {
    GuiShellAction::OpenDocument(OpenDocumentArgs {
        full_document_name: path.to_string(),
        visible: true,
        defer_content: false,
    })
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/open_dialog.rs"]
mod tests;
