// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::application::file_system::FolderEntry;
use std::time::Duration;
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
        if path != "/work/a.opd" {
            return Err(format!("file {path:?}: expected /work/a.opd"));
        }
        Ok(FileFacts {
            size: 12,
            created: Some(SystemTime::UNIX_EPOCH),
            modified: Some(SystemTime::UNIX_EPOCH + Duration::from_secs(90_061)),
            accessed: None,
        })
    }
}
fn value<'a>(fields: &'a [(&'static str, String)], label: &str) -> &'a str {
    &fields.iter().find(|(name, _)| *name == label).unwrap().1
}
#[test]
fn a_saved_document_reports_its_file_name_location_size_and_times() {
    let row = DocumentInfo {
        name: "a".into(),
        type_: "Part".into(),
        ..Default::default()
    };
    let facts = FakeFolders.file_facts("/work/a.opd");
    let fields = general_fields(Some(&row), Some("/work/a.opd"), &facts);
    assert_eq!(value(&fields, "File name"), "a.opd");
    assert_eq!(value(&fields, "Location"), "/work");
    assert_eq!(value(&fields, "Size"), "12 bytes");
    assert_eq!(value(&fields, "Modified"), "1970-01-02 01:01:01 UTC");
    assert_eq!(value(&fields, "Accessed"), NOT_AVAILABLE);
}
#[test]
fn an_unsaved_document_says_what_it_does_not_have_yet() {
    let fields = general_fields(
        Some(&DocumentInfo::default()),
        None,
        &Ok(FileFacts::default()),
    );
    assert_eq!(value(&fields, "Location"), NOT_SAVED);
    assert_eq!(value(&fields, "Size"), NOT_AVAILABLE);
}
#[test]
fn an_unreadable_file_shows_the_failure_and_not_a_false_size() {
    let facts = FakeFolders.file_facts("/gone.opd");
    let fields = general_fields(None, Some("/gone.opd"), &facts);
    assert!(value(&fields, "Size").contains("/gone.opd"));
}
#[test]
fn a_request_reads_the_facts_once_and_keeps_the_active_document_unchanged() {
    let mut properties = DocumentProperties::with_folders(Rc::new(FakeFolders));
    let mut state = GuiCommandState {
        document: Some(1),
        ..Default::default()
    };
    state.document_storage.insert(
        2,
        crate::GuiDocumentStorage {
            path: Some("/work/a.opd".into()),
            unsaved_edits: false,
        },
    );
    properties.request(&state, 2);
    assert!(properties.needs_frame());
    let target = properties.target.as_ref().unwrap();
    assert_eq!(target.id, 2);
    assert_eq!(target.facts.as_ref().unwrap().size, 12);
}
