// SPDX-License-Identifier: GPL-2.0-only

//! Ordered project reference and template lookup, independent of workspace ownership.
use oblikovati_api::types::*;
use std::path::{Component, Path, PathBuf};
mod attachments;
mod probe;
pub use attachments::ProjectAttachmentFiles;
pub use probe::{NativeProjectFileProbe, ProjectFileProbe, ProjectLookupError};

/// Validated absolute project roots. Lookup never depends on the process working directory.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProjectFileLookup {
    workspace: Option<PathBuf>,
    libraries: Vec<PathBuf>,
    templates: Option<PathBuf>,
}

impl ProjectFileLookup {
    /// Bind ordered roots; directories may be unavailable until lookup time.
    /// Example: `ProjectFileLookup::new(Some(root), vec![library], Some(templates))?`.
    pub fn new(
        workspace: Option<PathBuf>,
        libraries: Vec<PathBuf>,
        templates: Option<PathBuf>,
    ) -> Result<Self, ProjectLookupError> {
        Ok(Self {
            workspace: workspace.map(absolute_root).transpose()?,
            libraries: libraries
                .into_iter()
                .map(absolute_root)
                .collect::<Result<_, _>>()?,
            templates: templates.map(absolute_root).transpose()?,
        })
    }

    /// Inspect search priority without file access; example: `lookup.search_paths().collect::<Vec<_>>()`.
    pub fn search_paths(&self) -> impl Iterator<Item = &Path> {
        self.workspace
            .iter()
            .chain(&self.libraries)
            .map(PathBuf::as_path)
    }

    /// Resolve an absolute name or search workspace then libraries; missing files return `None`.
    /// Example: `let file = lookup.resolve(Path::new("pin.opd"), probe)?;`.
    pub fn resolve(
        &self,
        name: &Path,
        probe: &mut dyn ProjectFileProbe,
    ) -> Result<Option<PathBuf>, ProjectLookupError> {
        if name.as_os_str().is_empty() {
            return Ok(None);
        }
        if name.is_absolute() {
            return probe_candidate(name.to_owned(), probe);
        }
        for root in self.search_paths() {
            if let Some(found) = probe_candidate(root.join(name), probe)? {
                return Ok(Some(found));
            }
        }
        Ok(None)
    }

    /// Produce a lexical relative reference inside the workspace, without resolving symlinks.
    /// Example: `assert_eq!(lookup.relativize(&root.join("pin.opd")), Some("pin.opd".into()));`.
    pub fn relativize(&self, full_path: &Path) -> Option<PathBuf> {
        if !full_path.is_absolute() {
            return None;
        }
        let normalized = lexical_path(full_path);
        let relative = normalized.strip_prefix(self.workspace.as_ref()?).ok()?;
        Some(if relative.as_os_str().is_empty() {
            PathBuf::from(".")
        } else {
            relative.into()
        })
    }

    /// Find the standard template for a known API document kind; absent templates return `None`.
    /// Example: `let template = lookup.template_file(DOCUMENT_PART, probe)?;`.
    pub fn template_file(
        &self,
        kind: DocumentType,
        probe: &mut dyn ProjectFileProbe,
    ) -> Result<Option<PathBuf>, ProjectLookupError> {
        let Some(root) = &self.templates else {
            return Ok(None);
        };
        let Some((name, extension)) = template_name(kind) else {
            return Ok(None);
        };
        probe_candidate(root.join(format!("{name}{extension}")), probe)
    }
}

fn absolute_root(path: PathBuf) -> Result<PathBuf, ProjectLookupError> {
    if !path.is_absolute() {
        return Err(ProjectLookupError::RelativeRoot(path));
    }
    Ok(lexical_path(&path))
}

fn lexical_path(path: &Path) -> PathBuf {
    let mut normalized = PathBuf::new();
    for component in path.components() {
        match component {
            Component::CurDir => {}
            Component::ParentDir => {
                normalized.pop();
            }
            _ => normalized.push(component.as_os_str()),
        }
    }
    normalized
}

fn template_name(kind: DocumentType) -> Option<(&'static str, &'static str)> {
    match kind {
        DOCUMENT_PART => Some(("part", PART_FILE_EXTENSION)),
        DOCUMENT_ASSEMBLY => Some(("assembly", ASSEMBLY_FILE_EXTENSION)),
        DOCUMENT_DRAWING => Some(("drawing", DRAWING_FILE_EXTENSION)),
        DOCUMENT_PRESENTATION => Some(("presentation", PRESENTATION_FILE_EXTENSION)),
        _ => None,
    }
}

fn probe_candidate(
    path: PathBuf,
    probe: &mut dyn ProjectFileProbe,
) -> Result<Option<PathBuf>, ProjectLookupError> {
    match probe.is_regular_file(&path) {
        Ok(true) => Ok(Some(path)),
        Ok(false) => Ok(None),
        Err(cause) => Err(ProjectLookupError::Probe { path, cause }),
    }
}
