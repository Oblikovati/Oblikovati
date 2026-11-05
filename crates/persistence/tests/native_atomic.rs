// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::atomic::{AtomicFiles, NativeAtomicFiles, StagedFile, replace_file};
use std::{fs, path::Path};

#[test]
fn native_create_and_replace_leave_only_the_complete_destination() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("part.obk");
    for contents in [b"first".as_slice(), b"replacement\0\xff", b""] {
        replace_file(&destination, contents, &mut NativeAtomicFiles).unwrap();
        assert_eq!(fs::read(&destination).unwrap(), contents);
        assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
    }
}

#[test]
fn staged_bytes_do_not_touch_the_old_file_and_drop_removes_the_temporary() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("part.obk");
    fs::write(&destination, b"old").unwrap();
    let mut stage = NativeAtomicFiles.stage(&destination).unwrap();
    stage.write_contents(b"new").unwrap();
    stage.synchronize().unwrap();
    assert_eq!(fs::read(&destination).unwrap(), b"old");
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 2);
    drop(stage);
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
}

#[test]
fn failed_native_commit_cleans_the_temporary_and_retains_the_destination() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("occupied");
    fs::create_dir(&destination).unwrap();
    fs::write(destination.join("marker"), b"old").unwrap();
    let error = replace_file(&destination, b"new", &mut NativeAtomicFiles).unwrap_err();
    assert_eq!(error.phase, "commit");
    assert_eq!(fs::read(destination.join("marker")).unwrap(), b"old");
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
}

#[test]
fn invalid_destination_or_missing_parent_fails_before_staging() {
    assert!(NativeAtomicFiles.stage(Path::new("")).is_err());
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("missing").join("part.obk");
    let error = replace_file(&destination, b"new", &mut NativeAtomicFiles).unwrap_err();
    assert_eq!(error.phase, "stage");
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 0);
}

#[cfg(unix)]
#[test]
fn unix_document_permissions_match_the_legacy_writer() {
    use std::os::unix::fs::PermissionsExt;
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("part.obk");
    replace_file(&destination, b"new", &mut NativeAtomicFiles).unwrap();
    assert_eq!(
        fs::metadata(destination).unwrap().permissions().mode() & 0o777,
        0o644
    );
}
