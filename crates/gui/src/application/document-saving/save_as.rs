// SPDX-License-Identifier: GPL-2.0-only
use super::folder_browser::{
    ALL_FILES, DialogEnd, DialogEscape, FolderBrowser, dialog_buttons, file_name_field,
    file_type_field, full_path, proposed_file_name,
};
use crate::application::file_system::{GuiFolders, NativeFolders};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_api::wire::{CloseDocumentArgs, SaveDocumentAsArgs};
use std::rc::Rc;

const TITLE: &str = "Save As";
const IMAGE_CHOOSER: &str = "Select preview image";
const NO_PREVIEW: &str = "No preview image";
const CAPTURED_ON_SAVE: &str = "Picture captured on save";

/// Where the preview picture would come from when the document is saved.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum PreviewSource {
    #[default]
    ActiveComponentIso,
    ActiveWindow,
    ImportFromFile,
}
impl PreviewSource {
    const ALL: [Self; 3] = [
        Self::ActiveComponentIso,
        Self::ActiveWindow,
        Self::ImportFromFile,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::ActiveComponentIso => "Active Component Iso View on Save",
            Self::ActiveWindow => "Active Window on Save",
            Self::ImportFromFile => "Import From File",
        }
    }
}
struct SaveAsDraft {
    document: u64,
    close_after: bool,
    name: String,
    file_type: usize,
    save_preview: bool,
    source: PreviewSource,
    image: Option<String>,
    image_name: String,
    images: FolderBrowser,
    image_escape: DialogEscape,
    escape: DialogEscape,
    opening: bool,
}
pub(crate) struct SaveAsDialog {
    folders: Rc<dyn GuiFolders>,
    browser: FolderBrowser,
    draft: Option<SaveAsDraft>,
}
impl Default for SaveAsDialog {
    fn default() -> Self {
        Self::with_folders(Rc::new(NativeFolders))
    }
}
impl SaveAsDialog {
    pub(crate) fn with_folders(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            browser: FolderBrowser::new(folders.clone()),
            folders,
            draft: None,
        }
    }
    /// Open for a document, proposing its name; example: `dialog.request(7, "Part", false)`.
    pub(crate) fn request(&mut self, document: u64, name: &str, close_after: bool) {
        self.browser.refresh();
        self.draft = Some(SaveAsDraft {
            document,
            close_after,
            name: proposed_file_name(name),
            file_type: 0,
            save_preview: true,
            source: PreviewSource::default(),
            image: None,
            image_name: String::new(),
            images: FolderBrowser::new(self.folders.clone()),
            image_escape: DialogEscape::default(),
            escape: DialogEscape::default(),
            opening: true,
        });
    }
    /// A frame must follow so the popup opens without waiting for an event.
    pub(crate) fn is_opening(&self) -> bool {
        self.draft.as_ref().is_some_and(|draft| draft.opening)
    }
    pub(crate) fn is_open(&self) -> bool {
        self.draft.is_some()
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        let Some(draft) = &self.draft else {
            return Ok(());
        };
        if draft.name.contains('\0') || self.browser.folder.contains('\0') {
            return Err(format!(
                "save dialog name {:?}, folder {:?}: expected text without NUL",
                draft.name, self.browser.folder
            ));
        }
        Ok(())
    }
    /// Close a dialog whose document is gone; example: `dialog.synchronize(&state)`.
    fn synchronize(&mut self, state: &GuiCommandState) {
        let gone = self
            .draft
            .as_ref()
            .is_some_and(|draft| !state.documents.iter().any(|row| row.id == draft.document));
        if gone {
            self.draft = None;
        }
    }
    /// Draw the dialog; a returned end means it closed this frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<DialogEnd> {
        self.synchronize(state);
        let draft = self.draft.as_mut()?;
        if std::mem::take(&mut draft.opening) {
            ui.open_popup(TITLE);
        }
        dialog_geometry(ui);
        let _dialog = ui.begin_modal_popup(TITLE)?;
        let end = self.contents(ui, state, actions);
        if end.is_some() {
            self.draft = None;
            ui.close_current_popup();
        }
        end
    }
    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<DialogEnd> {
        let draft = self.draft.as_mut()?;
        ui.columns(2, "save-as-layout", true);
        let chosen = self.browser.controls(ui, "Save in", draft.file_type);
        if let Some(name) = chosen {
            draft.name = name;
        }
        ui.next_column();
        preview_options(ui, draft);
        ui.columns(1, "save-as-layout-end", false);
        ui.separator();
        footer(ui, state, &self.browser.folder, draft, actions)
    }
}
fn dialog_geometry(ui: &Ui) {
    let display = ui.io().display_size;
    let size = [860_f32.min(display[0] - 24.), 440_f32.min(display[1] - 24.)];
    let position = std::array::from_fn(|axis| (display[axis] - size[axis]) * 0.5);
    crate::widget_abi::window_geometry(ui, position, size, Condition::Appearing);
}
/// What the preview area shows; the picture itself is not stored in the file yet.
fn preview_text(draft: &SaveAsDraft) -> &str {
    if !draft.save_preview {
        return NO_PREVIEW;
    }
    match (draft.source, &draft.image) {
        (PreviewSource::ImportFromFile, Some(image)) => image,
        (PreviewSource::ImportFromFile, None) => NO_PREVIEW,
        _ => CAPTURED_ON_SAVE,
    }
}
fn preview_options(ui: &Ui, draft: &mut SaveAsDraft) {
    ui.child_window("##save-preview")
        .size([0., 110.])
        .border(true)
        .build(|| ui.text_disabled(preview_text(draft)));
    ui.checkbox("Save Preview Picture", &mut draft.save_preview);
    if draft.save_preview {
        source_choices(ui, draft);
    }
}
fn source_choices(ui: &Ui, draft: &mut SaveAsDraft) {
    for source in PreviewSource::ALL {
        if ui.radio_button_bool(source.label(), draft.source == source) {
            draft.source = source;
        }
    }
    if draft.source != PreviewSource::ImportFromFile {
        return;
    }
    ui.same_line();
    if crate::widget_abi::button(ui, "Browse...", [0.; 2]) {
        draft.images.refresh();
        ui.open_popup(IMAGE_CHOOSER);
    }
    image_chooser(ui, draft);
}
fn image_chooser(ui: &Ui, draft: &mut SaveAsDraft) {
    let Some(_chooser) = ui.begin_modal_popup(IMAGE_CHOOSER) else {
        return;
    };
    if let Some(name) = draft.images.controls(ui, "Look in", ALL_FILES) {
        draft.image_name = name;
    }
    let entered = file_name_field(ui, &mut draft.image_name);
    let can_open = !draft.image_name.is_empty();
    let end = dialog_buttons(ui, "Open", can_open, entered, &mut draft.image_escape);
    if end == Some(DialogEnd::Accepted) {
        draft.image = Some(full_path(&draft.images.folder, &draft.image_name));
    }
    if end.is_some() {
        ui.close_current_popup();
    }
}
fn footer(
    ui: &Ui,
    state: &GuiCommandState,
    folder: &str,
    draft: &mut SaveAsDraft,
    actions: &mut Vec<GuiShellAction>,
) -> Option<DialogEnd> {
    let entered = file_name_field(ui, &mut draft.name);
    file_type_field(ui, &mut draft.file_type);
    let action = GuiShellAction::SaveDocumentAs(SaveDocumentAsArgs {
        document: draft.document,
        new_full_document_name: full_path(folder, &draft.name),
    });
    let can_save = !draft.name.is_empty() && action.dispatchable(state);
    let end = dialog_buttons(ui, "Save", can_save, entered, &mut draft.escape);
    if end == Some(DialogEnd::Accepted) {
        actions.push(action);
        actions.extend(
            draft
                .close_after
                .then_some(close_after_save(draft.document)),
        );
    }
    end
}
/// Unforced on purpose: if the save fails, the document is still modified and the owner keeps it.
fn close_after_save(id: u64) -> GuiShellAction {
    GuiShellAction::CloseDocument(CloseDocumentArgs { id, force: false })
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/save_as.rs"]
mod tests;
