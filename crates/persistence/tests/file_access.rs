// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::file_access::{DocumentFiles, NativeDocumentFiles};
use std::{fs, path::Path};

#[test]
fn save_destination_uses_existing_parent_and_existing_file_canonicalization() {
    let directory = tempfile::tempdir().unwrap();
    let file = directory.path().join("new part.obk");
    let mut native = NativeDocumentFiles;
    assert_eq!(
        native.save_path(&file).unwrap(),
        fs::canonicalize(directory.path())
            .unwrap()
            .join("new part.obk")
    );
    fs::write(&file, b"original").unwrap();
    assert_eq!(
        native.save_path(&file).unwrap(),
        native.canonical_path(&file).unwrap()
    );
    assert_eq!(fs::read(&file).unwrap(), b"original");
}

#[test]
fn directory_missing_parent_and_empty_destination_are_refused() {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeDocumentFiles;
    assert!(
        native
            .save_path(directory.path())
            .unwrap_err()
            .to_string()
            .contains("regular file")
    );
    assert!(
        native
            .save_path(&directory.path().join("missing").join("part.obk"))
            .is_err()
    );
    assert!(
        native
            .save_path(Path::new(""))
            .unwrap_err()
            .to_string()
            .contains("file name")
    );
}

#[cfg(unix)]
#[test]
fn dangling_symlink_is_not_treated_as_an_unused_destination() {
    let directory = tempfile::tempdir().unwrap();
    let link = directory.path().join("link.obk");
    std::os::unix::fs::symlink(directory.path().join("missing.obk"), &link).unwrap();
    assert!(NativeDocumentFiles.save_path(&link).is_err());
    assert!(
        fs::symlink_metadata(&link)
            .unwrap()
            .file_type()
            .is_symlink()
    );
}
