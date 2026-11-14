// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::application::file_system::FileFacts;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PRESENTATION};
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
    fn system_locations(&self) -> Vec<(String, String)> {
        vec![("Home".to_string(), "/home".to_string())]
    }
    fn file_facts(&self, path: &str) -> Result<FileFacts, String> {
        Err(format!("file {path:?}: expected no file read"))
    }
}
fn entry(name: &str, is_folder: bool) -> FolderEntry {
    FolderEntry {
        name: name.into(),
        is_folder,
        size: 12,
        modified: None,
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
    let parts = FileFilter::Extension("opd");
    assert!(listed(&entry("old", true), parts));
    assert!(listed(&entry("a.opd", false), parts));
    assert!(listed(&entry("A.OPD", false), parts));
    assert!(!listed(&entry("b.png", false), parts));
    assert!(listed(&entry("b.png", false), FileFilter::Everything));
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
    assert_eq!(saving.filter(0), FileFilter::Extension("opd"));
    assert_eq!(saving.filter(1), FileFilter::Everything);
    assert_eq!(FileTypeList::opening().rows.len(), 9);
    assert_eq!(
        FileTypeList::assemblies().filter(0),
        FileFilter::Extension("oad")
    );
    assert_eq!(FileTypeList::any().filter(0), FileFilter::Everything);
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
fn visited_folders_appear_once_most_recent_first_and_a_reset_restores_the_recent_pane() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    browser.refresh();
    browser.navigate("old");
    browser.back();
    assert_eq!(browser.recent, ["/work/parts", "/work/parts/old"]);
    browser.navigation = NavigationPane::SystemLocations;
    assert_eq!(browser.navigation_rows()[0].0, "Home");
    browser.reset();
    assert_eq!(browser.navigation, NavigationPane::RecentFolders);
    assert_eq!(
        browser.navigation_rows()[0],
        ("parts".to_string(), "/work/parts".to_string())
    );
}
#[test]
fn a_file_type_names_its_document_kind() {
    use file_rows::entry_type;
    assert_eq!(entry_type(&entry("old", true)), "Folder");
    assert_eq!(entry_type(&entry("a.opd", false)), "Part");
    assert_eq!(entry_type(&entry("a.odd", false)), "Drawing");
    assert_eq!(entry_type(&entry("a.png", false)), "File");
}
#[test]
fn the_status_names_a_selection_or_counts_the_rows() {
    assert_eq!(status_text(2, 3), "2 items selected");
    assert_eq!(status_text(1, 3), "1 item selected");
    assert_eq!(status_text(0, 3), "3 items");
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
#[test]
fn the_open_list_starts_with_all_documents_and_offers_exchange_formats() {
    let types = FileTypeList::opening();
    assert!(types.labels()[0].starts_with("All Documents"));
    assert!(types.labels().iter().any(|label| label.starts_with("STEP")));
    assert!(types.filter(0).keeps("a.ODD") && !types.filter(0).keeps("a.stp"));
    assert_eq!(types.filter(99), FileFilter::Everything);
}
#[test]
fn only_a_picture_view_asks_for_pictures_and_it_lists_files_alone() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    browser.refresh();
    assert!(
        browser
            .listed_for_pictures(FileFilter::Everything)
            .is_empty()
    );
    browser.view = FileListView::Tiles;
    assert_eq!(
        browser.listed_for_pictures(FileFilter::Extension("opd")),
        ["/work/parts/a.opd"]
    );
}
#[test]
fn each_picture_view_has_a_cell_and_the_row_views_have_none() {
    for (view, _) in FileListView::MENU {
        assert_eq!(
            file_rows::FileCell::of_view(view).is_some(),
            view.shows_pictures(),
            "{view:?}"
        );
    }
}
#[test]
fn showing_the_dialog_again_forgets_what_it_chose_and_read() {
    let mut browser = FolderBrowser::new(Rc::new(FakeFolders));
    let image = crate::GuiPreviewImage {
        extent: [1, 1],
        rgb: std::sync::Arc::new(vec![[1, 2, 3]]),
    };
    browser.note_file_picture("/work/parts/a.opd", Some(&image));
    browser.several = vec!["a".into(), "b".into()];
    browser.folder_highlight = Some("old".into());
    browser.begin_showing();
    assert!(browser.file_pictures.is_empty() && browser.several.is_empty());
    assert_eq!(browser.highlighted_folder(), None);
}
