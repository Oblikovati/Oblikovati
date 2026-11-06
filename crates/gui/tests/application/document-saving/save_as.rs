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
    fn system_locations(&self) -> Vec<(String, String)> {
        vec![("Home".to_string(), "/home".to_string())]
    }
    fn file_facts(&self, path: &str) -> Result<FileFacts, String> {
        Err(format!("file {path:?}: expected no file read"))
    }
}
fn requested(close_after: bool) -> SaveAsDialog {
    let mut dialog = SaveAsDialog::with_folders(Rc::new(FakeFolders));
    dialog.request(7, "Part", None, close_after);
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
fn an_assembly_proposes_its_own_file_type() {
    let mut dialog = SaveAsDialog::with_folders(Rc::new(FakeFolders));
    dialog.request(
        7,
        "Assembly1",
        Some(oblikovati_api::types::DOCUMENT_ASSEMBLY),
        false,
    );
    let draft = dialog.draft.as_ref().unwrap();
    assert_eq!(draft.name, "Assembly1.oad");
    assert_eq!(draft.types.filter(0), Some("oad"));
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
fn the_active_window_shows_a_placeholder_until_its_image_arrives() {
    let mut dialog = requested(false);
    let draft = dialog.draft.as_mut().unwrap();
    draft.source = PreviewSource::ActiveWindow;
    assert_eq!(preview_text(draft), NO_PREVIEW);
    assert!(shown_window_preview(draft).is_none());
    let image = crate::GuiPreviewImage {
        extent: [1, 1],
        rgb: std::sync::Arc::new(vec![[1, 2, 3]]),
    };
    dialog.receive_window_preview(image);
    let draft = dialog.draft.as_mut().unwrap();
    assert!(shown_window_preview(draft).is_some());
    draft.save_preview = false;
    assert!(shown_window_preview(draft).is_none());
}
#[test]
fn a_new_request_forgets_what_a_cancelled_dialog_changed() {
    let mut dialog = requested(false);
    let draft = dialog.draft.as_mut().unwrap();
    draft.save_preview = false;
    draft.source = PreviewSource::ActiveWindow;
    dialog.browser.folder = "/elsewhere".into();
    dialog.request(7, "Part", None, false);
    let draft = dialog.draft.as_ref().unwrap();
    assert!(draft.save_preview);
    assert_eq!(draft.source, PreviewSource::ActiveComponentIso);
    assert_eq!(dialog.browser.folder, "/work");
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
