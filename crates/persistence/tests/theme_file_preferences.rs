// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_atomic;
use fake_atomic::{FakeAtomicFiles, FakeDisk, FakeStage};
use oblikovati_model::theme::{ThemeDefinition, ThemeSelectionStore};
use oblikovati_persistence::{
    atomic::AtomicFiles, load::PackageFileReader, theme_file_preferences::ThemeFilePreferences,
    theme_preferences::ThemePreferenceFile,
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};
type ThemeDisk = Rc<RefCell<FakeDisk>>;
type StagedPaths = Rc<RefCell<Vec<PathBuf>>>;
struct RecordedThemeFiles {
    disk: ThemeDisk,
    staged: StagedPaths,
}
impl RecordedThemeFiles {
    fn new(bytes: &[u8]) -> Self {
        Self {
            disk: Rc::new(RefCell::new(FakeDisk {
                committed: bytes.to_vec(),
                ..Default::default()
            })),
            staged: Default::default(),
        }
    }
}
impl PackageFileReader for RecordedThemeFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        FakeAtomicFiles(self.disk.clone()).read(path)
    }
}
impl AtomicFiles for RecordedThemeFiles {
    type Stage = FakeStage;
    fn stage(&mut self, path: &Path) -> io::Result<Self::Stage> {
        self.staged.borrow_mut().push(path.into());
        FakeAtomicFiles(self.disk.clone()).stage(path)
    }
}
struct ThemeStorageFixture {
    store: ThemeFilePreferences<RecordedThemeFiles>,
    custom: ThemeDisk,
    selection: ThemeDisk,
    staged: StagedPaths,
}
fn selection_file() -> (ThemePreferenceFile<RecordedThemeFiles>, ThemeDisk) {
    let files = RecordedThemeFiles::new(b"activeTheme: Workshop\nfuture: retained\n");
    let disk = files.disk.clone();
    (
        ThemePreferenceFile::load(Path::new("config/preferences.yaml"), files).unwrap(),
        disk,
    )
}
fn storage_fixture() -> ThemeStorageFixture {
    let (selection_file, selection) = selection_file();
    let files = RecordedThemeFiles::new(b"older custom bytes");
    let custom = files.disk.clone();
    let staged = files.staged.clone();
    let sources = BTreeMap::from([(
        "Workshop".into(),
        PathBuf::from("config/themes/from-vendor.xml"),
    )]);
    let store = ThemeFilePreferences::new(selection_file, files, sources);
    ThemeStorageFixture {
        store,
        custom,
        selection,
        staged,
    }
}
fn edited_theme() -> ThemeDefinition {
    let mut theme = ThemeDefinition::light()
        .unwrap()
        .duplicate("Workshop")
        .unwrap();
    let value = oblikovati_api::types::Rgba {
        r: 1.,
        g: 0.,
        b: 0.,
        a: 1.,
    };
    theme
        .set_color(&oblikovati_api::types::TOKEN_CHROME_ACCENT, value)
        .unwrap();
    theme
}
#[test]
fn custom_save_reuses_the_loaded_filename_and_keeps_selection_preferences_intact() {
    let mut fixture = storage_fixture();
    let theme = edited_theme();
    let selection = fixture.selection.borrow().committed.clone();
    assert!(fixture.store.can_save_custom(&theme));
    fixture.store.save_custom_theme(&theme).unwrap();
    assert_eq!(
        *fixture.staged.borrow(),
        [PathBuf::from("config/themes/from-vendor.xml")]
    );
    let restored = ThemeDefinition::from_xml(
        &fixture.custom.borrow().committed,
        "fallback",
        oblikovati_api::types::THEME_CUSTOM,
    )
    .unwrap();
    assert_eq!(restored.name(), "Workshop");
    assert_eq!(restored.hex_colors(), theme.hex_colors());
    assert_eq!(fixture.selection.borrow().committed, selection);
    assert_eq!(fixture.store.active_name(), "Workshop");
}
#[test]
fn selection_save_uses_the_same_provider_without_rewriting_the_custom_xml() {
    let mut fixture = storage_fixture();
    let before = fixture.custom.borrow().committed.clone();
    fixture
        .store
        .save_selection(&ThemeDefinition::light().unwrap())
        .unwrap();
    assert_eq!(fixture.store.active_name(), "Light");
    assert_eq!(fixture.custom.borrow().committed, before);
    let preferences: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&fixture.selection.borrow().committed).unwrap();
    assert_eq!(
        preferences["future"],
        serde_yaml_ng::Value::String("retained".into())
    );
}
#[test]
fn unknown_custom_sources_refuse_before_any_staging() {
    let mut fixture = storage_fixture();
    let unknown = ThemeDefinition::dark()
        .unwrap()
        .duplicate("Missing")
        .unwrap();
    assert!(!fixture.store.can_save_custom(&unknown));
    let error = fixture.store.save_custom(&unknown).unwrap_err();
    assert!(error.contains("Missing") && error.contains("known loaded XML source"));
    assert!(fixture.staged.borrow().is_empty());
    assert_eq!(fixture.custom.borrow().committed, b"older custom bytes");
}
#[test]
fn failed_source_writes_preserve_the_previous_custom_file_and_active_selection() {
    for phase in ["stage", "write", "sync", "commit"] {
        let mut fixture = storage_fixture();
        fixture.custom.borrow_mut().fail_at = Some(phase);
        assert!(
            fixture
                .store
                .save_custom(&edited_theme())
                .unwrap_err()
                .contains(phase)
        );
        assert_eq!(fixture.custom.borrow().committed, b"older custom bytes");
        assert_eq!(fixture.store.active_name(), "Workshop");
    }
}

impl oblikovati_persistence::theme_lifecycle::ThemeLifecycleFiles for RecordedThemeFiles {
    fn create_theme(&mut self, _directory: &Path, _xml: &[u8]) -> io::Result<PathBuf> {
        Err(io::Error::other(
            "RecordedThemeFiles expects existing-source writes only",
        ))
    }
    fn remove_theme(&mut self, _source: &Path) -> io::Result<()> {
        Err(io::Error::other(
            "RecordedThemeFiles expects existing-source writes only",
        ))
    }
}
