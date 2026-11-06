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
    assert_eq!(proposed_file_name("Part", "opd"), "Part.opd");
    assert_eq!(proposed_file_name("Part.opd", "opd"), "Part.opd");
    assert_eq!(proposed_file_name("Assembly1", "oad"), "Assembly1.oad");
}
#[test]
fn a_filter_keeps_folders_and_its_files_only() {
    assert!(listed(&entry("old", true), Some("opd")));
    assert!(listed(&entry("a.opd", false), Some("opd")));
    assert!(listed(&entry("A.OPD", false), Some("opd")));
    assert!(!listed(&entry("b.png", false), Some("opd")));
    assert!(listed(&entry("b.png", false), None));
}
#[test]
fn each_document_kind_saves_as_its_native_type() {
    assert_eq!(native_file_type(Some(DOCUMENT_ASSEMBLY)).extension, "oad");
    assert_eq!(native_file_type(Some(DOCUMENT_DRAWING)).extension, "odd");
    assert_eq!(
        native_file_type(Some(DOCUMENT_PRESENTATION)).extension,
        "ord"
    );
    assert_eq!(native_file_type(None).extension, "opd");
}
#[test]
fn the_extensions_are_the_registered_ones() {
    use oblikovati_api::types::*;
    for (native, registered) in [
        (PART_FILES, PART_FILE_EXTENSION),
        (ASSEMBLY_FILES, ASSEMBLY_FILE_EXTENSION),
        (DRAWING_FILES, DRAWING_FILE_EXTENSION),
        (PRESENTATION_FILES, PRESENTATION_FILE_EXTENSION),
    ] {
        assert_eq!(format!(".{}", native.extension), registered);
        assert!(
            native.label.contains(registered),
            "label {:?}",
            native.label
        );
    }
}
#[test]
fn file_type_rows_name_what_they_keep() {
    let saving = FileTypeList::saving(PART_FILES);
    assert_eq!(saving.labels(), [PART_FILES.label, ALL_FILES]);
    assert_eq!(saving.filter(0), Some("opd"));
    assert_eq!(saving.filter(1), None);
    assert_eq!(FileTypeList::opening().rows.len(), 5);
    assert_eq!(FileTypeList::assemblies().filter(0), Some("oad"));
    assert_eq!(FileTypeList::any().filter(0), None);
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
fn back_returns_through_the_folders_left_behind() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    browser.navigate("old");
    browser.navigate("older");
    browser.back();
    assert_eq!(browser.folder, "/work/parts/old");
    browser.back();
    assert_eq!(browser.folder, "/work/parts");
    browser.back();
    assert_eq!(browser.folder, "/work/parts");
}
#[test]
fn a_reset_starts_over_in_the_initial_folder_with_no_history() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    browser.navigate("old");
    browser.selected = Some("a.opd".into());
    browser.reset();
    assert_eq!(browser.folder, "/work/parts");
    assert!(browser.selected.is_none() && browser.history.is_empty());
}
#[test]
fn the_status_names_a_selection_or_counts_the_rows() {
    assert_eq!(status_text(true, 3), "1 item selected");
    assert_eq!(status_text(false, 3), "3 items");
}
#[test]
fn the_breadcrumb_shows_each_step_of_the_folder() {
    assert_eq!(breadcrumb("/work/parts"), "/ > work > parts");
    assert_eq!(breadcrumb("/a/b/c/d/e"), "... > c > d > e");
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
