// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_atomic;
use fake_atomic::{FakeAtomicFiles, FakeDisk, FakeStage};
use oblikovati_model::theme::{ThemeDefinition, ThemeSelectionStore};
use oblikovati_persistence::{
    atomic::AtomicFiles,
    load::PackageFileReader,
    theme_file_preferences::ThemeFilePreferences,
    theme_lifecycle::ThemeLifecycleFiles,
    theme_preferences::{NativeThemePreferenceFiles, ThemePreferenceFile},
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};
#[derive(Default)]
struct ThemeSourceDisk {
    sources: BTreeMap<PathBuf, Vec<u8>>,
    fail_create: bool,
    fail_remove: bool,
}
struct LifecycleThemeFiles {
    disk: Rc<RefCell<ThemeSourceDisk>>,
    atomic: Rc<RefCell<FakeDisk>>,
}
impl PackageFileReader for LifecycleThemeFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        FakeAtomicFiles(self.atomic.clone()).read(path)
    }
}
impl AtomicFiles for LifecycleThemeFiles {
    type Stage = FakeStage;
    fn stage(&mut self, path: &Path) -> io::Result<Self::Stage> {
        FakeAtomicFiles(self.atomic.clone()).stage(path)
    }
}
impl ThemeLifecycleFiles for LifecycleThemeFiles {
    fn create_theme(&mut self, directory: &Path, xml: &[u8]) -> io::Result<PathBuf> {
        let mut disk = self.disk.borrow_mut();
        if disk.fail_create {
            return Err(io::Error::other("injected creation failure"));
        }
        let path = directory.join(format!("unique-{}.xml", disk.sources.len()));
        assert!(!disk.sources.contains_key(&path));
        disk.sources.insert(path.clone(), xml.into());
        Ok(path)
    }
    fn remove_theme(&mut self, source: &Path) -> io::Result<()> {
        let mut disk = self.disk.borrow_mut();
        if disk.fail_remove {
            return Err(io::Error::other("injected removal failure"));
        }
        disk.sources
            .remove(source)
            .ok_or_else(|| io::Error::from(io::ErrorKind::NotFound))?;
        Ok(())
    }
}
fn lifecycle_fixture() -> (
    ThemeFilePreferences<LifecycleThemeFiles>,
    Rc<RefCell<ThemeSourceDisk>>,
) {
    let disk = Rc::new(RefCell::new(ThemeSourceDisk::default()));
    let files = || LifecycleThemeFiles {
        disk: disk.clone(),
        atomic: Default::default(),
    };
    let selection =
        ThemePreferenceFile::load(Path::new("config/preferences.yaml"), files()).unwrap();
    let store = ThemeFilePreferences::new(selection, files(), BTreeMap::new())
        .with_custom_directory("config/themes".into());
    (store, disk)
}
fn custom(name: &str) -> ThemeDefinition {
    ThemeDefinition::dark().unwrap().duplicate(name).unwrap()
}
#[test]
fn creation_keeps_similar_names_distinct_and_deletion_uses_the_recorded_source() {
    let (mut store, disk) = lifecycle_fixture();
    for name in ["My Theme", "My-Theme", "../🌙"] {
        store.create_custom(&custom(name)).unwrap();
    }
    assert_eq!(disk.borrow().sources.len(), 3);
    for bytes in disk.borrow().sources.values() {
        let restored =
            ThemeDefinition::from_xml(bytes, "fallback", oblikovati_api::types::THEME_CUSTOM)
                .unwrap();
        assert!(store.can_save_custom(&restored));
    }
    store.remove_custom(&custom("My-Theme")).unwrap();
    assert!(!store.can_save_custom(&custom("My-Theme")));
    assert!(
        !disk
            .borrow()
            .sources
            .contains_key(Path::new("config/themes/unique-1.xml"))
    );
    assert_eq!(disk.borrow().sources.len(), 2);
}
#[test]
fn creation_refuses_builtins_and_duplicate_names_without_replacing_bytes() {
    let (mut store, disk) = lifecycle_fixture();
    store.create_custom(&custom("Workshop")).unwrap();
    let before = disk.borrow().sources.clone();
    for theme in [custom("Workshop"), ThemeDefinition::light().unwrap()] {
        let cause = store.create_custom(&theme).unwrap_err();
        assert!(cause.contains(theme.name()) && cause.contains("expected"));
    }
    assert_eq!(disk.borrow().sources, before);
}
#[test]
fn storage_failures_do_not_publish_or_discard_source_identity() {
    let (mut store, disk) = lifecycle_fixture();
    let theme = custom("Workshop");
    disk.borrow_mut().fail_create = true;
    assert!(
        store
            .create_custom(&theme)
            .unwrap_err()
            .contains("creation failure")
    );
    assert!(!store.can_save_custom(&theme));
    disk.borrow_mut().fail_create = false;
    store.create_custom(&theme).unwrap();
    disk.borrow_mut().fail_remove = true;
    assert!(
        store
            .remove_custom(&theme)
            .unwrap_err()
            .contains("removal failure")
    );
    assert!(store.can_save_custom(&theme));
    assert_eq!(disk.borrow().sources.len(), 1);
    for theme in [custom("Unknown"), ThemeDefinition::dark().unwrap()] {
        assert!(
            store
                .remove_custom(&theme)
                .unwrap_err()
                .contains(theme.name())
        );
    }
}
#[test]
fn absent_custom_directory_refuses_creation() {
    let files = || LifecycleThemeFiles {
        disk: Default::default(),
        atomic: Default::default(),
    };
    let selection = ThemePreferenceFile::load(Path::new("preferences.yaml"), files()).unwrap();
    let mut store = ThemeFilePreferences::new(selection, files(), BTreeMap::new());
    assert!(
        store
            .create_custom(&custom("Workshop"))
            .unwrap_err()
            .contains("configured custom directory")
    );
}
#[test]
fn native_creation_publishes_complete_unique_files_and_removes_only_the_requested_source() {
    let directory = tempfile::tempdir().unwrap();
    let root = directory.path().join("themes");
    let mut files = NativeThemePreferenceFiles;
    let first = files.create_theme(&root, b"first XML").unwrap();
    let second = files.create_theme(&root, b"second XML").unwrap();
    assert_ne!(first, second);
    assert_eq!(first.extension().unwrap(), "xml");
    assert_eq!(std::fs::read(&first).unwrap(), b"first XML");
    assert_eq!(std::fs::read(&second).unwrap(), b"second XML");
    files.remove_theme(&first).unwrap();
    assert!(!first.exists());
    assert!(second.exists());
    assert!(files.remove_theme(&first).is_err());
    assert!(files.create_theme(&second, b"blocked").is_err());
    assert_eq!(std::fs::read_dir(&root).unwrap().count(), 1);
}

#[test]
fn deletion_uses_an_imported_vendor_filename_without_deriving_a_slug() {
    let disk = Rc::new(RefCell::new(ThemeSourceDisk::default()));
    let source = PathBuf::from("config/themes/from-vendor.xml");
    disk.borrow_mut()
        .sources
        .insert(source.clone(), b"vendor XML".to_vec());
    let files = || LifecycleThemeFiles {
        disk: disk.clone(),
        atomic: Default::default(),
    };
    let selection =
        ThemePreferenceFile::load(Path::new("config/preferences.yaml"), files()).unwrap();
    let mut store = ThemeFilePreferences::new(
        selection,
        files(),
        BTreeMap::from([("Workshop".into(), source)]),
    );
    store.remove_custom_theme(&custom("Workshop")).unwrap();
    assert!(disk.borrow().sources.is_empty());
    assert!(!store.can_save_custom(&custom("Workshop")));
}
