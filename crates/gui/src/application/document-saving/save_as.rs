// SPDX-License-Identifier: GPL-2.0-only
use super::folder_browser::{
    DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field, file_type_field, full_path, native_file_type, proposed_file_name,
};
use crate::application::file_system::{GuiFolders, NativeFolders};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::{
    types::DocumentType,
    wire::{CloseDocumentArgs, SaveDocumentAsArgs},
};
use std::rc::Rc;

const TITLE: &str = "Save As";
const IMAGE_CHOOSER: &str = "Select preview image";
const NO_PREVIEW: &str = "No preview image";
const CAPTURED_ON_SAVE: &str = "Picture captured on save";
const PREVIEW_HEIGHT: f32 = 110.;

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
    types: FileTypeList,
    file_type: usize,
    save_preview: bool,
    source: PreviewSource,
    image: Option<String>,
    image_name: String,
    window_preview: Option<crate::GuiPreviewImage>,
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
    /// Open for a document, proposing its name; example: `dialog.request(7, "Part", None, false)`.
    pub(crate) fn request(
        &mut self,
        document: u64,
        name: &str,
        kind: Option<DocumentType>,
        close_after: bool,
    ) {
        let native = native_file_type(kind);
        self.browser.reset();
        self.draft = Some(SaveAsDraft {
            document,
            close_after,
            name: proposed_file_name(name, native.extension),
            types: FileTypeList::saving(native),
            file_type: 0,
            save_preview: true,
            source: PreviewSource::default(),
            image: None,
            image_name: String::new(),
            window_preview: None,
            images: FolderBrowser::new(self.folders.clone()),
            image_escape: DialogEscape::default(),
            escape: DialogEscape::default(),
            opening: true,
        });
    }
    /// Show the image the owner captured for the active window.
    pub(crate) fn receive_window_preview(&mut self, image: crate::GuiPreviewImage) {
        if let Some(draft) = &mut self.draft {
            draft.window_preview = Some(image);
        }
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
        center_dialog(ui, [860., 440.]);
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
        let filter = draft.types.filter(draft.file_type);
        let chosen = self.browser.controls(ui, "Save in", filter);
        if let Some(name) = chosen {
            draft.name = name;
        }
        ui.next_column();
        preview_options(ui, draft, actions);
        ui.columns(1, "save-as-layout-end", false);
        ui.separator();
        footer(ui, state, &self.browser.folder, draft, actions)
    }
}
/// What the preview area shows; the picture itself is not stored in the file yet.
fn preview_text(draft: &SaveAsDraft) -> &str {
    if !draft.save_preview {
        return NO_PREVIEW;
    }
    match (draft.source, &draft.image) {
        (PreviewSource::ImportFromFile, Some(image)) => image,
        (PreviewSource::ImportFromFile | PreviewSource::ActiveWindow, _) => NO_PREVIEW,
        _ => CAPTURED_ON_SAVE,
    }
}
fn shown_window_preview(draft: &SaveAsDraft) -> Option<&crate::GuiPreviewImage> {
    (draft.save_preview && draft.source == PreviewSource::ActiveWindow)
        .then_some(draft.window_preview.as_ref())
        .flatten()
}
/// A grid of filled cells: the GUI binds one font texture, and a preview needs no other.
fn draw_preview_image(ui: &Ui, image: &crate::GuiPreviewImage) {
    let origin = ui.cursor_screen_pos();
    let size = ui.content_region_avail();
    let cell = (size[0] / image.extent[0] as f32).min(size[1] / image.extent[1] as f32);
    let list = ui.get_window_draw_list();
    for (index, color) in image.rgb.iter().enumerate() {
        let x = origin[0] + (index as u32 % image.extent[0]) as f32 * cell;
        let y = origin[1] + (index as u32 / image.extent[0]) as f32 * cell;
        let rgba = [
            f32::from(color[0]) / 255.,
            f32::from(color[1]) / 255.,
            f32::from(color[2]) / 255.,
            1.,
        ];
        list.add_rect([x, y], [x + cell, y + cell], rgba)
            .filled(true)
            .build();
    }
}
fn preview_options(ui: &Ui, draft: &mut SaveAsDraft, actions: &mut Vec<GuiShellAction>) {
    ui.child_window("##save-preview")
        .size([0., PREVIEW_HEIGHT])
        .border(true)
        .build(|| match shown_window_preview(draft) {
            Some(image) => draw_preview_image(ui, image),
            None => ui.text_disabled(preview_text(draft)),
        });
    ui.checkbox("Save Preview Picture", &mut draft.save_preview);
    if draft.save_preview {
        source_choices(ui, draft, actions);
    }
}
fn source_choices(ui: &Ui, draft: &mut SaveAsDraft, actions: &mut Vec<GuiShellAction>) {
    for source in PreviewSource::ALL {
        if ui.radio_button_bool(source.label(), draft.source == source) {
            draft.source = source;
        }
        if source == draft.source {
            source_control(ui, draft, actions);
        }
    }
    image_chooser(ui, draft);
}
/// The action beside a source that needs one: a capture now, or an image to choose.
fn source_control(ui: &Ui, draft: &mut SaveAsDraft, actions: &mut Vec<GuiShellAction>) {
    match draft.source {
        PreviewSource::ActiveWindow => {
            ui.same_line();
            if crate::widget_abi::button(ui, "Capture", [0.; 2]) {
                actions.push(GuiShellAction::CaptureWindowPreview);
            }
        }
        PreviewSource::ImportFromFile => {
            ui.same_line();
            if crate::widget_abi::button(ui, "Browse...", [0.; 2]) {
                draft.images.refresh();
                ui.open_popup(IMAGE_CHOOSER);
            }
        }
        PreviewSource::ActiveComponentIso => {}
    }
}
fn image_chooser(ui: &Ui, draft: &mut SaveAsDraft) {
    let Some(_chooser) = ui.begin_modal_popup(IMAGE_CHOOSER) else {
        return;
    };
    if let Some(name) = draft
        .images
        .controls(ui, "Look in", FileTypeList::any().filter(0))
    {
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
    file_type_field(ui, &draft.types, &mut draft.file_type);
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
