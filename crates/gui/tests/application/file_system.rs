// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::time::Duration;
fn row(name: &str, is_folder: bool) -> FolderEntry {
    FolderEntry {
        name: name.into(),
        is_folder,
        size: 0,
        modified: None,
    }
}
#[test]
fn folders_come_first_and_names_sort_within_each_kind() {
    let mut rows = vec![
        row("b.opd", false),
        row("z", true),
        row("a.opd", false),
        row("c", true),
    ];
    sort_entries(&mut rows);
    let names: Vec<&str> = rows.iter().map(|entry| entry.name.as_str()).collect();
    assert_eq!(names, ["c", "z", "a.opd", "b.opd"]);
}
#[test]
fn native_folders_read_a_real_listing_and_real_facts() {
    let folder = tempfile::tempdir().unwrap();
    std::fs::create_dir(folder.path().join("inner")).unwrap();
    let file = folder.path().join("part.opd");
    std::fs::write(&file, b"12345").unwrap();
    let rows = NativeFolders
        .entries(folder.path().to_str().unwrap())
        .unwrap();
    let names: Vec<(&str, bool)> = rows
        .iter()
        .map(|entry| (entry.name.as_str(), entry.is_folder))
        .collect();
    assert_eq!(names, [("inner", true), ("part.opd", false)]);
    assert_eq!(rows[1].size, 5);
    assert!(rows[1].modified.is_some());
    let facts = NativeFolders.file_facts(file.to_str().unwrap()).unwrap();
    assert_eq!(facts.size, 5);
    assert!(facts.modified.is_some());
}
#[test]
fn an_unreadable_folder_or_file_names_the_offending_path() {
    assert!(
        NativeFolders
            .entries("/no/such/folder")
            .unwrap_err()
            .contains("/no/such/folder")
    );
    assert!(
        NativeFolders
            .file_facts("/no/such/file")
            .unwrap_err()
            .contains("/no/such/file")
    );
}
#[test]
fn the_system_pane_always_offers_the_computer_root() {
    let places = NativeFolders.system_locations();
    assert!(
        places
            .iter()
            .any(|(label, folder)| label == "Computer" && folder == "/")
    );
}
#[test]
fn the_calendar_matches_known_dates() {
    assert_eq!(civil_date(0), (1970, 1, 1));
    assert_eq!(civil_date(-1), (1969, 12, 31));
    assert_eq!(civil_date(11_016), (2000, 2, 29));
    assert_eq!(civil_date(20_763), (2026, 11, 6));
}
#[test]
fn a_timestamp_reads_in_utc() {
    let stamp = SystemTime::UNIX_EPOCH + Duration::from_secs(90_061);
    assert_eq!(timestamp_text(stamp), "1970-01-02 01:01:01 UTC");
}
