// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::orientation_preferences::OrientationPreferenceAccess;
use oblikovati_persistence::orientation_preferences::OrientationPreferenceFile;
use std::{cell::RefCell, path::Path, rc::Rc};

#[test]
fn orientation_lock_round_trips_and_preserves_other_preferences() {
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"compassHidden: true\n".to_vec(),
        ..Default::default()
    }));
    let load = || {
        OrientationPreferenceFile::load(
            Path::new("ui-preferences.yaml"),
            FakeAtomicFiles(disk.clone()),
        )
        .unwrap()
    };
    let mut preferences = load();
    assert!(!preferences.orientation_lock());
    preferences.set_orientation_lock(true).unwrap();
    assert!(load().orientation_lock());
    let saved = String::from_utf8(disk.borrow().committed.clone()).unwrap();
    assert!(saved.contains("compassHidden: true"));
    preferences.set_orientation_lock(false).unwrap();
    assert!(!load().orientation_lock());
}

#[test]
fn failed_preference_write_does_not_publish_a_new_value() {
    let disk = Rc::new(RefCell::new(FakeDisk::default()));
    let mut preferences =
        OrientationPreferenceFile::load(Path::new("prefs.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    disk.borrow_mut().fail_at = Some("commit");
    assert!(preferences.set_orientation_lock(true).is_err());
    assert!(!preferences.orientation_lock());
    assert!(disk.borrow().committed.is_empty());
}

#[test]
fn invalid_saved_lock_is_reported_without_overwriting_the_file() {
    let original = b"lockToSelection: invalid\n";
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: original.to_vec(),
        ..Default::default()
    }));
    let error = OrientationPreferenceFile::load(
        Path::new("ui-preferences.yaml"),
        FakeAtomicFiles(disk.clone()),
    )
    .err()
    .expect("invalid lock");
    assert!(error.to_string().contains("ui-preferences.yaml"));
    assert_eq!(disk.borrow().committed, original);
}
