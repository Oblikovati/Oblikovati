// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::attachment::{ExternalAttachmentFiles, ResolvedAttachmentSource};
use oblikovati_persistence::project::{ProjectAttachmentFiles, ProjectFileLookup};
use std::{io, time::SystemTime};
#[path = "support/project_paths.rs"]
mod project_paths;
use project_paths::native_project_fixture_root as root;

fn fixture_file(directory: &str) -> String {
    let path = root(directory).join("source.bin");
    path.to_str()
        .unwrap_or_else(|| panic!("fixture path {path:?}: expected a UTF-8 native path"))
        .to_owned()
}

#[derive(Default)]
struct SearchFiles {
    present: Vec<String>,
    inspected: Vec<String>,
    read_paths: Vec<String>,
    denied: Option<String>,
}
impl ExternalAttachmentFiles for SearchFiles {
    fn inspect_source(&mut self, path: &str) -> io::Result<Option<ResolvedAttachmentSource>> {
        self.inspected.push(path.into());
        if self.denied.as_deref() == Some(path) {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "source denied",
            ));
        }
        Ok(self
            .present
            .iter()
            .any(|entry| entry == path)
            .then(|| ResolvedAttachmentSource {
                resolved_file_name: path.into(),
                modified: SystemTime::UNIX_EPOCH,
            }))
    }
    fn read(&mut self, path: &str) -> io::Result<Vec<u8>> {
        self.read_paths.push(path.into());
        Ok(vec![0, 255])
    }
}
fn project() -> ProjectFileLookup {
    ProjectFileLookup::new(Some(root("workspace")), vec![root("library")], None).unwrap()
}

#[test]
fn ordered_search_retains_successful_observation_without_second_probe() {
    let lookup = project();
    let mut files = SearchFiles {
        present: vec![fixture_file("library")],
        ..Default::default()
    };
    let source = ProjectAttachmentFiles::new(&lookup, &mut files)
        .inspect_source("source.bin")
        .unwrap()
        .unwrap();
    assert_eq!(source.resolved_file_name, fixture_file("library"));
    assert_eq!(source.modified, SystemTime::UNIX_EPOCH);
    assert_eq!(
        files.inspected,
        [fixture_file("workspace"), fixture_file("library")]
    );
    files.present.push(fixture_file("workspace"));
    files.inspected.clear();
    let source = ProjectAttachmentFiles::new(&lookup, &mut files)
        .inspect_source("source.bin")
        .unwrap()
        .unwrap();
    assert_eq!(source.resolved_file_name, fixture_file("workspace"));
    assert_eq!(files.inspected, [fixture_file("workspace")]);
}

#[test]
fn payload_read_uses_selected_path_and_missing_read_names_reference() {
    let lookup = project();
    let mut files = SearchFiles {
        present: vec![fixture_file("library")],
        ..Default::default()
    };
    assert_eq!(
        ProjectAttachmentFiles::new(&lookup, &mut files)
            .read("source.bin")
            .unwrap(),
        [0, 255]
    );
    assert_eq!(files.read_paths, [fixture_file("library")]);
    files.present.clear();
    let error = ProjectAttachmentFiles::new(&lookup, &mut files)
        .read("source.bin")
        .unwrap_err();
    assert_eq!(error.kind(), io::ErrorKind::NotFound);
    assert!(error.to_string().contains("source.bin"));
    assert_eq!(files.read_paths.len(), 1);
}

#[test]
fn absolute_empty_and_unconfigured_references_do_not_search_other_roots() {
    let lookup = project();
    let mut files = SearchFiles::default();
    assert!(
        ProjectAttachmentFiles::new(&lookup, &mut files)
            .inspect_source(&fixture_file("elsewhere"))
            .unwrap()
            .is_none()
    );
    assert_eq!(files.inspected, [fixture_file("elsewhere")]);
    files.inspected.clear();
    assert!(
        ProjectAttachmentFiles::new(&lookup, &mut files)
            .inspect_source("")
            .unwrap()
            .is_none()
    );
    let empty = ProjectFileLookup::new(None, vec![], None).unwrap();
    assert!(
        ProjectAttachmentFiles::new(&empty, &mut files)
            .inspect_source("source.bin")
            .unwrap()
            .is_none()
    );
    assert!(files.inspected.is_empty());
}

#[test]
fn access_error_stops_search_and_preserves_cause() {
    let lookup = project();
    let mut files = SearchFiles {
        denied: Some(fixture_file("workspace")),
        present: vec![fixture_file("library")],
        ..Default::default()
    };
    let error = ProjectAttachmentFiles::new(&lookup, &mut files)
        .inspect_source("source.bin")
        .unwrap_err();
    assert_eq!(error.kind(), io::ErrorKind::PermissionDenied);
    assert!(
        error
            .to_string()
            .contains(&format!("{:?}", root("workspace").join("source.bin")))
    );
    assert!(error.to_string().contains("source denied"));
    assert_eq!(files.inspected, [fixture_file("workspace")]);
}

#[cfg(unix)]
#[test]
fn non_utf8_root_is_refused_before_foreign_io() {
    use std::{ffi::OsString, os::unix::ffi::OsStringExt, path::PathBuf};
    let root = PathBuf::from(OsString::from_vec(b"/bad\xff".to_vec()));
    let lookup = ProjectFileLookup::new(Some(root), vec![], None).unwrap();
    let mut files = SearchFiles::default();
    let error = ProjectAttachmentFiles::new(&lookup, &mut files)
        .inspect_source("source.bin")
        .unwrap_err();
    assert_eq!(error.kind(), io::ErrorKind::InvalidInput);
    assert!(error.to_string().contains("UTF-8"));
    assert!(files.inspected.is_empty());
}
