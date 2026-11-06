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
#[test]
fn opening_keeps_the_path_exactly_and_shows_the_document() {
    let GuiShellAction::OpenDocument(request) = open_action(" /tmp/部品.opd ") else {
        panic!("expected an OpenDocument action");
    };
    assert_eq!(request.full_document_name, " /tmp/部品.opd ");
    assert!(request.visible && !request.defer_content);
}
#[test]
fn a_request_starts_empty_and_asks_for_a_frame() {
    let mut dialog = OpenDialog::with_folders(Rc::new(FakeFolders));
    assert!(!dialog.is_opening());
    dialog.request();
    assert!(dialog.is_opening());
    assert_eq!(dialog.draft.as_ref().unwrap().name, "");
}
#[test]
fn nul_in_the_name_is_refused_with_the_value() {
    let mut dialog = OpenDialog::with_folders(Rc::new(FakeFolders));
    dialog.validate().unwrap();
    dialog.request();
    dialog.draft.as_mut().unwrap().name = "bad\0path".into();
    assert!(dialog.validate().unwrap_err().contains("bad\\0path"));
}
