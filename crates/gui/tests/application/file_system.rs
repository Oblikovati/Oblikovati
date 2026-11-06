// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn row(name: &str, is_folder: bool) -> FolderEntry {
    FolderEntry {
        name: name.into(),
        is_folder,
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
    assert_eq!(rows, [row("inner", true), row("part.opd", false)]);
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
