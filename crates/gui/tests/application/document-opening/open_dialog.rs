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
#[test]
fn opening_keeps_the_path_exactly_and_shows_the_document() {
    let GuiShellAction::OpenDocument(request) = open_action(" /tmp/部品.opd ", false) else {
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
#[test]
fn several_names_read_in_quotes() {
    let names = vec!["a.odd".to_string(), "b.odd".to_string()];
    assert_eq!(quoted_names(&names), "\"a.odd\" \"b.odd\"");
}
#[test]
fn what_is_open_opens_first_so_a_closed_file_ends_in_front() {
    let browser = FolderBrowser::new(Rc::new(FakeFolders));
    let mut state = GuiCommandState::default();
    state.document_storage.insert(
        1,
        crate::GuiDocumentStorage {
            path: Some("/work/b.odd".into()),
            ..Default::default()
        },
    );
    let names = vec!["a.odd".to_string(), "b.odd".to_string()];
    assert_eq!(
        in_opening_order(&browser, &names, &state),
        ["/work/b.odd", "/work/a.odd"]
    );
}
#[test]
fn inspections_ask_for_one_native_file_at_a_time_and_ignore_old_answers() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    let mut inspections = FileInspections::default();
    let mut actions = Vec::new();
    let wanted = || {
        ["/w/notes.txt", "/w/a.opd", "/w/b.opd"]
            .map(str::to_owned)
            .into_iter()
    };
    inspections.ask_next(wanted(), &mut actions);
    inspections.ask_next(wanted(), &mut actions);
    assert_eq!(actions, [GuiShellAction::InspectFile("/w/a.opd".into())]);
    let answer = |path: &str| GuiCommandState {
        inspected_file: Some(GuiInspectedFile {
            path: path.into(),
            ..Default::default()
        }),
        ..Default::default()
    };
    inspections.follow(&answer("/w/old.opd"), &mut browser);
    assert!(inspections.of("/w/old.opd").is_none());
    inspections.follow(&answer("/w/a.opd"), &mut browser);
    assert!(inspections.of("/w/a.opd").is_some());
    inspections.ask_next(wanted(), &mut actions);
    assert_eq!(actions[1], GuiShellAction::InspectFile("/w/b.opd".into()));
}
