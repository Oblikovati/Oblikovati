// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::application::file_system::{FileFacts, FolderEntry};
struct FakeFolders;
impl GuiFolders for FakeFolders {
    fn default_folder(&self) -> String {
        "/work".into()
    }
    fn entries(&self, _: &str) -> Result<Vec<FolderEntry>, String> {
        Ok(Vec::new())
    }
    fn file_facts(&self, path: &str) -> Result<FileFacts, String> {
        Err(format!("file {path:?}: expected no file read"))
    }
}
fn requested(close_after: bool) -> SaveAsDialog {
    let mut dialog = SaveAsDialog::with_folders(Rc::new(FakeFolders));
    dialog.request(7, "Part", close_after);
    dialog
}
#[test]
fn a_request_proposes_the_name_and_the_default_preview() {
    let dialog = requested(true);
    let draft = dialog.draft.as_ref().unwrap();
    assert_eq!(draft.name, "Part.opd");
    assert!(draft.close_after && draft.save_preview);
    assert_eq!(draft.source, PreviewSource::ActiveComponentIso);
    assert!(dialog.is_opening() && dialog.is_open());
}
#[test]
fn the_preview_area_follows_the_options() {
    let mut dialog = requested(false);
    let draft = dialog.draft.as_mut().unwrap();
    assert_eq!(preview_text(draft), CAPTURED_ON_SAVE);
    draft.source = PreviewSource::ImportFromFile;
    assert_eq!(preview_text(draft), NO_PREVIEW);
    draft.image = Some("/work/a.png".into());
    assert_eq!(preview_text(draft), "/work/a.png");
    draft.save_preview = false;
    assert_eq!(preview_text(draft), NO_PREVIEW);
}
#[test]
fn nul_in_a_field_is_refused_with_the_value() {
    let mut dialog = requested(false);
    dialog.draft.as_mut().unwrap().name = "a\0b".into();
    assert!(dialog.validate().unwrap_err().contains("a\\0b"));
}
#[test]
fn a_dialog_closes_when_its_document_is_gone() {
    let mut dialog = requested(false);
    dialog.synchronize(&GuiCommandState::default());
    assert!(!dialog.is_open());
}
#[test]
fn the_close_after_a_save_is_never_forced() {
    assert_eq!(
        close_after_save(7),
        GuiShellAction::CloseDocument(CloseDocumentArgs {
            id: 7,
            force: false
        })
    );
}
