// SPDX-License-Identifier: GPL-2.0-only
use super::*;
struct FakeAddInDirectoryReader {
    entries: Vec<(&'static str, bool)>,
    failure: Option<io::ErrorKind>,
}
impl AddInDirectoryReader for FakeAddInDirectoryReader {
    fn entries(&self, _: &Path) -> io::Result<Vec<AddInDirectoryEntry>> {
        if let Some(kind) = self.failure {
            return Err(io::Error::new(kind, "named directory fault"));
        }
        Ok(self
            .entries
            .iter()
            .map(|(name, directory)| AddInDirectoryEntry {
                name: name.into(),
                directory: *directory,
            })
            .collect())
    }
}
#[test]
fn flat_discovery_sorts_shared_library_names_and_ignores_directories() {
    let reader = FakeAddInDirectoryReader {
        entries: vec![
            ("z.SO", false),
            ("nested.so", true),
            ("notes.txt", false),
            ("b.dll", false),
            ("a.dylib", false),
        ],
        failure: None,
    };
    assert_eq!(
        discover_addins(&reader, Path::new("/extensions")).unwrap(),
        ["a.dylib", "b.dll", "z.SO"].map(|name| Path::new("/extensions").join(name))
    );
    assert!(!shared_library_name(OsStr::new("no-extension")));
}
#[test]
fn missing_directory_is_empty_and_other_io_errors_are_named() {
    let reader = FakeAddInDirectoryReader {
        entries: vec![],
        failure: Some(io::ErrorKind::NotFound),
    };
    assert!(
        discover_addins(&reader, Path::new("/missing"))
            .unwrap()
            .is_empty()
    );
    let denied = FakeAddInDirectoryReader {
        failure: Some(io::ErrorKind::PermissionDenied),
        ..reader
    };
    assert!(
        discover_addins(&denied, Path::new("/denied"))
            .unwrap_err()
            .contains("/denied")
    );
}
#[test]
fn discovery_directory_preserves_go_override_and_executable_rules() {
    assert_eq!(
        addin_directory(Some(OsStr::new("/chosen")), Ok("/bin/head".into())),
        PathBuf::from("/chosen")
    );
    assert_eq!(
        addin_directory(Some(OsStr::new("")), Ok("/bin/head".into())),
        PathBuf::from("/bin/addins")
    );
    assert_eq!(
        addin_directory(None, Err(io::Error::other("missing executable"))),
        PathBuf::from("addins")
    );
}
#[test]
fn native_directory_reader_reports_real_names_and_types() {
    let directory = tempfile::tempdir().unwrap();
    std::fs::write(directory.path().join("module.so"), b"fixture").unwrap();
    std::fs::create_dir(directory.path().join("nested.so")).unwrap();
    let paths = discover_addins(&NativeAddInDirectoryReader, directory.path()).unwrap();
    assert_eq!(paths, [directory.path().join("module.so")]);
}
