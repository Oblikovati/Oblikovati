// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::{
    marking_menu::{MarkingMenuPreferenceAccess, MarkingMenus},
    theme::ThemeDefinition,
};
use oblikovati_persistence::{
    marking_menu_preferences::MarkingMenuPreferenceFile, theme_preferences::ThemePreferenceFile,
};
use std::{cell::RefCell, path::Path, rc::Rc};
#[test]
fn marking_menu_does_not_overwrite_new_ambiguous_field() {
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"menus: []\n".to_vec(),
        ..Default::default()
    }));
    let mut file =
        MarkingMenuPreferenceFile::load(Path::new("menu.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    let changed = b"menus: []\nfuture: {'<<': {x: 1}}\n";
    disk.borrow_mut().committed = changed.to_vec();
    assert!(file.save(&MarkingMenus::default()).is_err());
    assert_eq!(disk.borrow().committed, changed);
}
#[test]
fn theme_does_not_overwrite_new_ambiguous_field() {
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"activeTheme: Light\n".to_vec(),
        ..Default::default()
    }));
    let mut file =
        ThemePreferenceFile::load(Path::new("theme.yaml"), FakeAtomicFiles(disk.clone())).unwrap();
    let changed = b"activeTheme: Light\nfuture: {'<<': {x: 1}}\n";
    disk.borrow_mut().committed = changed.to_vec();
    assert!(
        file.save_active(&ThemeDefinition::light().unwrap())
            .is_err()
    );
    assert_eq!(disk.borrow().committed, changed);
}

#[test]
fn keymap_does_not_overwrite_external_edit() {
    use oblikovati_api::wire::KeymapExport;
    use oblikovati_model::keymap::KeymapPreferenceAccess;
    use oblikovati_persistence::keymap_preferences::KeymapPreferenceFile;
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"chords: {Open: Ctrl+O}\n".to_vec(),
        ..Default::default()
    }));
    let mut file =
        KeymapPreferenceFile::load(Path::new("keys.yaml"), FakeAtomicFiles(disk.clone())).unwrap();
    let previous = file.keymap();
    let changed = b"chords: {Open: Ctrl+P}\n";
    disk.borrow_mut().committed = changed.to_vec();
    assert!(file.set_keymap(&KeymapExport::default()).is_err());
    assert_eq!(file.keymap(), previous);
    assert_eq!(disk.borrow().committed, changed);
    assert!(!disk.borrow().calls.contains(&"stage"));
}

#[test]
fn missing_preference_created_after_load_is_not_overwritten() {
    use std::io;
    let disk = Rc::new(RefCell::new(FakeDisk {
        read_error: Some(io::ErrorKind::NotFound),
        ..Default::default()
    }));
    let mut file =
        ThemePreferenceFile::load(Path::new("theme.yaml"), FakeAtomicFiles(disk.clone())).unwrap();
    disk.borrow_mut().read_error = None;
    assert!(
        file.save_active(&ThemeDefinition::light().unwrap())
            .is_err()
    );
    assert!(!disk.borrow().calls.contains(&"stage"));
}

#[test]
fn successive_owned_saves_update_the_observed_revision() {
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"activeTheme: Light\n".to_vec(),
        ..Default::default()
    }));
    let mut file =
        ThemePreferenceFile::load(Path::new("theme.yaml"), FakeAtomicFiles(disk)).unwrap();
    file.save_active(&ThemeDefinition::dark().unwrap()).unwrap();
    file.save_active(&ThemeDefinition::light().unwrap())
        .unwrap();
}
