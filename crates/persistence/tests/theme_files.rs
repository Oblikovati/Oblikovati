// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_atomic;
use fake_atomic::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::theme::ThemeDefinition;
use oblikovati_persistence::{
    load::PackageFileReader,
    theme_files::{ThemeDirectoryFiles, load_custom_themes, save_custom_theme},
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    ffi::OsString,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};
#[derive(Default)]
struct FakeThemeDirectory {
    entries: Vec<OsString>,
    contents: BTreeMap<PathBuf, Vec<u8>>,
    reads: Vec<PathBuf>,
    directory_error: Option<io::ErrorKind>,
}
impl ThemeDirectoryFiles for FakeThemeDirectory {
    fn theme_entries(&mut self, _: &Path) -> io::Result<Vec<OsString>> {
        if let Some(kind) = self.directory_error {
            return Err(io::Error::new(kind, "injected theme directory decline"));
        }
        Ok(self.entries.clone())
    }
}
impl PackageFileReader for FakeThemeDirectory {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.reads.push(path.into());
        self.contents
            .get(path)
            .cloned()
            .ok_or_else(|| io::Error::new(io::ErrorKind::NotFound, "injected missing theme file"))
    }
}
impl FakeThemeDirectory {
    fn insert(&mut self, name: &str, bytes: Vec<u8>) {
        self.entries.push(name.into());
        self.contents
            .insert(Path::new("config/themes").join(name), bytes);
    }
}
fn custom_xml(name: &str) -> Vec<u8> {
    ThemeDefinition::dark()
        .unwrap()
        .duplicate(name)
        .unwrap()
        .to_xml()
        .unwrap()
}
#[test]
fn custom_gallery_loads_in_filename_order_with_plain_blender_fallback_names() {
    let mut files = FakeThemeDirectory::default();
    files.insert("z.xml", custom_xml("Workshop"));
    files.insert(
        "a.xml",
        include_bytes!("../../model/assets/themes/light.xml").to_vec(),
    );
    files.insert("ignored.XML", b"bad".to_vec());
    let loaded = load_custom_themes(Path::new("config"), &mut files).unwrap();
    assert!(loaded.declines.is_empty());
    assert_eq!(
        loaded
            .customs
            .iter()
            .map(|theme| theme.name())
            .collect::<Vec<_>>(),
        ["a", "Workshop"]
    );
    assert_eq!(files.reads.len(), 2);
    assert_eq!(loaded.sources["a"], Path::new("config/themes/a.xml"));
    assert_eq!(loaded.sources["Workshop"], Path::new("config/themes/z.xml"));
}
#[test]
fn malformed_missing_duplicate_and_builtin_names_decline_without_losing_good_files() {
    let mut files = FakeThemeDirectory::default();
    files.insert("a.xml", custom_xml("Workshop"));
    files.insert("b.xml", custom_xml("Workshop"));
    files.insert("builtin.xml", custom_xml("Dark"));
    files.insert("broken.xml", b"<broken>".to_vec());
    files.entries.push("missing.xml".into());
    files.entries.push("../escape.xml".into());
    let loaded = load_custom_themes(Path::new("config"), &mut files).unwrap();
    assert_eq!(loaded.customs.len(), 1);
    assert_eq!(loaded.customs[0].name(), "Workshop");
    assert_eq!(loaded.sources["Workshop"], Path::new("config/themes/a.xml"));
    assert_eq!(loaded.sources.len(), 1);
    assert_eq!(loaded.declines.len(), 5);
    assert!(
        loaded
            .declines
            .iter()
            .all(|decline| !decline.path.as_os_str().is_empty()
                && decline.cause.contains("expected"))
    );
    assert!(!files.reads.iter().any(|path| path.ends_with("escape.xml")));
}
#[test]
fn directory_errors_are_reported_and_an_empty_directory_yields_no_customs() {
    let mut files = FakeThemeDirectory::default();
    assert!(
        load_custom_themes(Path::new("config"), &mut files)
            .unwrap()
            .customs
            .is_empty()
    );
    files.directory_error = Some(io::ErrorKind::PermissionDenied);
    assert!(load_custom_themes(Path::new("config"), &mut files).is_err());
    assert!(files.reads.is_empty());
}
#[test]
fn custom_export_round_trips_a_complete_palette_and_refuses_builtins() {
    let disk = Rc::new(RefCell::new(FakeDisk::default()));
    let mut files = FakeAtomicFiles(disk.clone());
    let theme = ThemeDefinition::light()
        .unwrap()
        .duplicate("Workshop")
        .unwrap();
    save_custom_theme(Path::new("workshop.xml"), &theme, &mut files).unwrap();
    let restored = ThemeDefinition::from_xml(
        &disk.borrow().committed,
        "fallback",
        oblikovati_api::types::THEME_CUSTOM,
    )
    .unwrap();
    assert_eq!(restored.name(), "Workshop");
    assert_eq!(restored.hex_colors(), theme.hex_colors());
    let before = disk.borrow().committed.clone();
    assert!(
        save_custom_theme(
            Path::new("dark.xml"),
            &ThemeDefinition::dark().unwrap(),
            &mut files
        )
        .is_err()
    );
    assert_eq!(disk.borrow().committed, before);
}
#[test]
fn failed_custom_export_keeps_the_prior_file_at_each_atomic_phase() {
    for phase in ["stage", "write", "sync", "commit"] {
        let disk = Rc::new(RefCell::new(FakeDisk {
            committed: b"old".to_vec(),
            fail_at: Some(phase),
            ..Default::default()
        }));
        let theme = ThemeDefinition::dark()
            .unwrap()
            .duplicate("Workshop")
            .unwrap();
        let error = save_custom_theme(
            Path::new("workshop.xml"),
            &theme,
            &mut FakeAtomicFiles(disk.clone()),
        )
        .unwrap_err();
        assert!(error.contains(phase));
        assert_eq!(disk.borrow().committed, b"old");
    }
}

#[test]
fn dot_xml_uses_a_snapshot_name_but_refuses_an_unnamed_blender_export() {
    let mut files = FakeThemeDirectory::default();
    files.insert(".xml", custom_xml("Hidden"));
    let loaded = load_custom_themes(Path::new("config"), &mut files).unwrap();
    assert_eq!(loaded.customs[0].name(), "Hidden");
    files.contents.insert(
        Path::new("config/themes/.xml").into(),
        include_bytes!("../../model/assets/themes/dark.xml").to_vec(),
    );
    let loaded = load_custom_themes(Path::new("config"), &mut files).unwrap();
    assert!(loaded.customs.is_empty());
    assert_eq!(loaded.declines.len(), 1);
    assert!(loaded.declines[0].cause.contains("nonempty"));
}
#[cfg(unix)]
#[test]
fn non_utf8_xml_names_decline_before_file_reads() {
    use std::os::unix::ffi::OsStringExt;
    let mut files = FakeThemeDirectory {
        entries: vec![OsString::from_vec(vec![255, b'.', b'x', b'm', b'l'])],
        ..Default::default()
    };
    let loaded = load_custom_themes(Path::new("config"), &mut files).unwrap();
    assert!(loaded.customs.is_empty());
    assert_eq!(loaded.declines.len(), 1);
    assert!(loaded.declines[0].cause.contains("UTF-8"));
    assert!(files.reads.is_empty());
}
