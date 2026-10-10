// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_persistence::project::*;
use std::{
    collections::BTreeSet,
    io,
    path::{Path, PathBuf},
};

#[derive(Default)]
struct RecordingFileProbe {
    files: BTreeSet<PathBuf>,
    visited: Vec<PathBuf>,
    denied: Option<PathBuf>,
}
impl ProjectFileProbe for RecordingFileProbe {
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool> {
        self.visited.push(path.into());
        if self.denied.as_deref() == Some(path) {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "project locked",
            ));
        }
        Ok(self.files.contains(path))
    }
}

fn root(name: &str) -> PathBuf {
    std::env::temp_dir()
        .join("oblikovati-project-fixtures")
        .join(name)
}
fn project() -> ProjectFileLookup {
    ProjectFileLookup::new(
        Some(root("workspace")),
        vec![root("lib1"), root("lib2")],
        Some(root("templates")),
    )
    .unwrap()
}

#[test]
fn reference_search_uses_workspace_then_libraries_and_stops_at_first_match() {
    let lookup = project();
    assert_eq!(
        lookup.search_paths().collect::<Vec<_>>(),
        [root("workspace"), root("lib1"), root("lib2")]
    );
    let mut probe = RecordingFileProbe::default();
    probe.files.insert(root("lib2/pin.opd"));
    assert_eq!(
        lookup.resolve(Path::new("pin.opd"), &mut probe).unwrap(),
        Some(root("lib2/pin.opd"))
    );
    assert_eq!(
        probe.visited,
        [
            root("workspace/pin.opd"),
            root("lib1/pin.opd"),
            root("lib2/pin.opd")
        ]
    );
    probe.files.insert(root("workspace/pin.opd"));
    probe.visited.clear();
    assert_eq!(
        lookup.resolve(Path::new("pin.opd"), &mut probe).unwrap(),
        Some(root("workspace/pin.opd"))
    );
    assert_eq!(probe.visited, [root("workspace/pin.opd")]);
}

#[test]
fn absolute_names_bypass_search_roots_and_missing_files_are_not_errors() {
    let lookup = project();
    let mut probe = RecordingFileProbe::default();
    let absolute = root("external/pin.opd");
    probe.files.insert(absolute.clone());
    assert_eq!(
        lookup.resolve(&absolute, &mut probe).unwrap(),
        Some(absolute.clone())
    );
    assert_eq!(probe.visited, [absolute]);
    assert!(
        lookup
            .resolve(&root("missing.opd"), &mut probe)
            .unwrap()
            .is_none()
    );
    assert!(
        lookup
            .resolve(Path::new("missing.opd"), &mut probe)
            .unwrap()
            .is_none()
    );
    let count = probe.visited.len();
    assert!(lookup.resolve(Path::new(""), &mut probe).unwrap().is_none());
    assert_eq!(probe.visited.len(), count);
}

#[test]
fn inaccessible_workspace_is_reported_instead_of_silently_selecting_a_library() {
    let mut probe = RecordingFileProbe {
        denied: Some(root("workspace/pin.opd")),
        ..Default::default()
    };
    probe.files.insert(root("lib1/pin.opd"));
    let error = project()
        .resolve(Path::new("pin.opd"), &mut probe)
        .unwrap_err();
    assert!(error.to_string().contains("project locked"));
    assert!(error.to_string().contains("pin.opd"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(probe.visited, [root("workspace/pin.opd")]);
}

#[test]
fn templates_use_canonical_document_extensions_and_unknown_or_missing_return_none() {
    let lookup = project();
    let mut probe = RecordingFileProbe::default();
    for (kind, name) in [
        (DOCUMENT_PART, "part.opd"),
        (DOCUMENT_ASSEMBLY, "assembly.oad"),
        (DOCUMENT_DRAWING, "drawing.odd"),
        (DOCUMENT_PRESENTATION, "presentation.ord"),
    ] {
        let expected = root("templates").join(name);
        probe.files.insert(expected.clone());
        assert_eq!(
            lookup.template_file(kind, &mut probe).unwrap(),
            Some(expected)
        );
    }
    let visited = probe.visited.len();
    assert!(
        lookup
            .template_file(DOCUMENT_UNKNOWN, &mut probe)
            .unwrap()
            .is_none()
    );
    assert!(
        lookup
            .template_file(DocumentType(999), &mut probe)
            .unwrap()
            .is_none()
    );
    assert_eq!(probe.visited.len(), visited);
    probe.files.clear();
    assert!(
        lookup
            .template_file(DOCUMENT_PART, &mut probe)
            .unwrap()
            .is_none()
    );
}

#[test]
fn relative_references_use_components_and_normalize_parent_segments() {
    let lookup = project();
    assert_eq!(
        lookup.relativize(&root("workspace/sub/pin.opd")),
        Some(PathBuf::from("sub/pin.opd"))
    );
    assert_eq!(
        lookup.relativize(&root("workspace/sub/../pin.opd")),
        Some(PathBuf::from("pin.opd"))
    );
    assert_eq!(
        lookup.relativize(&root("workspace/..cache/pin.opd")),
        Some(PathBuf::from("..cache/pin.opd"))
    );
    assert_eq!(
        lookup.relativize(&root("workspace")),
        Some(PathBuf::from("."))
    );
    assert!(
        lookup
            .relativize(&root("workspace/../outside.opd"))
            .is_none()
    );
    assert!(
        lookup
            .relativize(&root("workspace-other/pin.opd"))
            .is_none()
    );
    assert!(lookup.relativize(Path::new("relative.opd")).is_none());
}

#[test]
fn absent_roots_do_not_fall_back_to_the_process_current_directory() {
    let lookup = ProjectFileLookup::new(None, vec![], None).unwrap();
    let mut probe = RecordingFileProbe::default();
    assert_eq!(lookup.search_paths().count(), 0);
    assert!(
        lookup
            .resolve(Path::new("pin.opd"), &mut probe)
            .unwrap()
            .is_none()
    );
    assert!(
        lookup
            .template_file(DOCUMENT_PART, &mut probe)
            .unwrap()
            .is_none()
    );
    assert!(lookup.relativize(&root("pin.opd")).is_none());
    assert!(probe.visited.is_empty());
}

#[test]
fn every_configured_root_must_be_absolute() {
    for (workspace, libraries, templates) in [
        (Some(PathBuf::from("relative")), vec![], None),
        (None, vec![PathBuf::from("")], None),
        (None, vec![], Some(PathBuf::from("templates"))),
    ] {
        let error = ProjectFileLookup::new(workspace, libraries, templates).unwrap_err();
        assert!(
            error
                .to_string()
                .contains("expected an absolute directory path")
        );
        assert!(std::error::Error::source(&error).is_none());
    }
    let lookup = ProjectFileLookup::new(Some(root("workspace/sub/..")), vec![], None).unwrap();
    assert_eq!(
        lookup.search_paths().collect::<Vec<_>>(),
        [root("workspace")]
    );
}

#[test]
fn native_lookup_distinguishes_regular_files_directories_and_missing_components() {
    let directory = tempfile::tempdir().unwrap();
    let workspace = directory.path().join("workspace");
    let library = directory.path().join("library");
    std::fs::create_dir_all(&workspace).unwrap();
    std::fs::create_dir_all(&library).unwrap();
    let file = library.join("part.opd");
    std::fs::write(&file, b"recipe").unwrap();
    let lookup = ProjectFileLookup::new(
        Some(workspace.clone()),
        vec![library.clone()],
        Some(library),
    )
    .unwrap();
    let mut probe = NativeProjectFileProbe;
    assert_eq!(
        lookup.resolve(Path::new("part.opd"), &mut probe).unwrap(),
        Some(file.clone())
    );
    assert_eq!(
        lookup.template_file(DOCUMENT_PART, &mut probe).unwrap(),
        Some(file.clone())
    );
    assert!(lookup.resolve(&workspace, &mut probe).unwrap().is_none());
    assert!(
        lookup
            .resolve(&file.join("not-a-directory"), &mut probe)
            .unwrap()
            .is_none()
    );
    assert!(
        lookup
            .resolve(Path::new("missing.opd"), &mut probe)
            .unwrap()
            .is_none()
    );
}
