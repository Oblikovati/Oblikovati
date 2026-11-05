// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::theme::{ThemeDefinition, ThemeLibrary};
use oblikovati_persistence::theme_preferences::ThemePreferenceFile;
use std::{cell::RefCell, io, path::Path, rc::Rc};
type ThemeDisk = Rc<RefCell<FakeDisk>>;
fn theme_disk(encoded: &[u8]) -> ThemeDisk {
    Rc::new(RefCell::new(FakeDisk {
        committed: encoded.to_vec(),
        ..Default::default()
    }))
}
fn load(disk: &ThemeDisk) -> io::Result<ThemePreferenceFile<FakeAtomicFiles>> {
    ThemePreferenceFile::load(Path::new("preferences.yaml"), FakeAtomicFiles(disk.clone()))
}
#[test]
fn empty_missing_and_unknown_selections_use_the_library_dark_default() {
    for encoded in [b"".as_slice(), b"activeTheme: Missing\n"] {
        let disk = theme_disk(encoded);
        let preferences = load(&disk).unwrap();
        let library = ThemeLibrary::new(Vec::new(), preferences.active_name()).unwrap();
        assert_eq!(library.active().name(), "Dark");
    }
    let disk = theme_disk(b"");
    disk.borrow_mut().read_error = Some(io::ErrorKind::NotFound);
    assert_eq!(load(&disk).unwrap().active_name(), "");
}
#[test]
fn saved_selection_reopens_without_losing_other_go_preference_groups() {
    let disk = theme_disk(b"activeTheme: Dark\nfuture: {enabled: true}\n");
    let mut preferences = load(&disk).unwrap();
    preferences
        .save_active(&ThemeDefinition::light().unwrap())
        .unwrap();
    assert_eq!(preferences.active_name(), "Light");
    assert_eq!(load(&disk).unwrap().active_name(), "Light");
    let saved: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
    assert_eq!(saved["future"]["enabled"], serde_yaml_ng::Value::Bool(true));
}
#[test]
fn failed_atomic_saves_preserve_the_old_selection_in_memory_and_on_disk() {
    for phase in ["stage", "write", "sync", "commit"] {
        let disk = theme_disk(b"activeTheme: Dark\n");
        let mut preferences = load(&disk).unwrap();
        let before = disk.borrow().committed.clone();
        disk.borrow_mut().fail_at = Some(phase);
        let error = preferences
            .save_active(&ThemeDefinition::light().unwrap())
            .unwrap_err();
        assert!(error.to_string().contains(phase));
        assert_eq!(preferences.active_name(), "Dark");
        assert_eq!(disk.borrow().committed, before);
    }
}
#[test]
fn malformed_and_unreadable_preferences_are_reported_without_overwrite() {
    for encoded in ["activeTheme: [", "[1, 2]", "activeTheme: {name: Dark}"] {
        let disk = theme_disk(encoded.as_bytes());
        let error = load(&disk).err().unwrap();
        assert!(error.to_string().contains("preferences.yaml"));
        assert_eq!(disk.borrow().calls, ["read"]);
        assert_eq!(disk.borrow().committed, encoded.as_bytes());
    }
    let disk = theme_disk(b"activeTheme: Dark");
    disk.borrow_mut().read_error = Some(io::ErrorKind::PermissionDenied);
    assert!(load(&disk).is_err());
}
