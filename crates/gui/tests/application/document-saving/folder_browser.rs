// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::application::file_system::FileFacts;
struct FakeFolders;
impl GuiFolders for FakeFolders {
    fn default_folder(&self) -> String {
        "/work/parts".into()
    }
    fn entries(&self, folder: &str) -> Result<Vec<FolderEntry>, String> {
        if folder == "/work/parts" {
            return Ok(vec![
                entry("old", true),
                entry("a.opd", false),
                entry("b.png", false),
            ]);
        }
        Err(format!("folder {folder:?}: expected /work/parts"))
    }
    fn file_facts(&self, path: &str) -> Result<FileFacts, String> {
        Err(format!("file {path:?}: expected no file read"))
    }
}
fn entry(name: &str, is_folder: bool) -> FolderEntry {
    FolderEntry {
        name: name.into(),
        is_folder,
    }
}
#[test]
fn a_document_name_becomes_a_native_file_name_once() {
    assert_eq!(proposed_file_name("Part"), "Part.opd");
    assert_eq!(proposed_file_name("Part.opd"), "Part.opd");
}
#[test]
fn the_part_filter_keeps_folders_and_native_files_only() {
    assert!(listed(&entry("old", true), 0));
    assert!(listed(&entry("a.opd", false), 0));
    assert!(!listed(&entry("b.png", false), 0));
    assert!(listed(&entry("b.png", false), ALL_FILES));
}
#[test]
fn a_folder_row_reads_as_a_folder() {
    assert_eq!(row_label(&entry("old", true)), "old/");
    assert_eq!(row_label(&entry("a.opd", false)), "a.opd");
}
#[test]
fn a_typed_absolute_name_wins_over_the_folder() {
    assert_eq!(full_path("/work", "a.opd"), "/work/a.opd");
    assert_eq!(full_path("/work", "/other/a.opd"), "/other/a.opd");
}
#[test]
fn navigation_reads_the_new_folder_and_reports_a_bad_one() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    browser.refresh();
    assert_eq!(browser.listing.as_ref().unwrap().len(), 3);
    browser.navigate("old");
    assert_eq!(browser.folder, "/work/parts/old");
    assert!(
        browser
            .listing
            .as_ref()
            .unwrap_err()
            .contains("/work/parts/old")
    );
    browser.up();
    assert_eq!(browser.folder, "/work/parts");
    assert_eq!(browser.listing.as_ref().unwrap().len(), 3);
}
