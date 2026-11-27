// SPDX-License-Identifier: GPL-2.0-only

use super::{ProjectFileLookup, ProjectFileProbe, ProjectLookupError};
use oblikovati_model::attachment::{ExternalAttachmentFiles, ResolvedAttachmentSource};
use std::{io, path::Path};

/// Project search order applied to foreign attachment metadata and payload reads.
pub struct ProjectAttachmentFiles<'a> {
    lookup: &'a ProjectFileLookup,
    files: &'a mut dyn ExternalAttachmentFiles,
}

impl<'a> ProjectAttachmentFiles<'a> {
    /// Bind a project and foreign-file provider without accessing any files.
    /// Example: `ProjectAttachmentFiles::new(&project, &mut native)`.
    pub fn new(lookup: &'a ProjectFileLookup, files: &'a mut dyn ExternalAttachmentFiles) -> Self {
        Self { lookup, files }
    }
}

impl ExternalAttachmentFiles for ProjectAttachmentFiles<'_> {
    fn inspect_source(&mut self, path: &str) -> io::Result<Option<ResolvedAttachmentSource>> {
        let mut probe = AttachmentCandidate {
            files: self.files,
            found: None,
        };
        self.lookup
            .resolve(Path::new(path), &mut probe)
            .map_err(lookup_error)?;
        Ok(probe.found)
    }

    fn read(&mut self, path: &str) -> io::Result<Vec<u8>> {
        let source = self.inspect_source(path)?.ok_or_else(|| {
            io::Error::new(
                io::ErrorKind::NotFound,
                format!(
                    "attachment source {path:?}: expected a regular file in project search paths"
                ),
            )
        })?;
        self.files.read(&source.resolved_file_name)
    }
}

struct AttachmentCandidate<'a> {
    files: &'a mut dyn ExternalAttachmentFiles,
    found: Option<ResolvedAttachmentSource>,
}
impl ProjectFileProbe for AttachmentCandidate<'_> {
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool> {
        let path = path.to_str().ok_or_else(|| {
            io::Error::new(
                io::ErrorKind::InvalidInput,
                format!("attachment candidate {path:?}: expected a UTF-8 source path"),
            )
        })?;
        // Retain the successful observation so the caller never re-probes it (#3693).
        self.found = self.files.inspect_source(path)?;
        Ok(self.found.is_some())
    }
}

fn lookup_error(error: ProjectLookupError) -> io::Error {
    let kind = match &error {
        ProjectLookupError::Probe { cause, .. } => cause.kind(),
        ProjectLookupError::RelativeRoot(_) => io::ErrorKind::InvalidInput,
    };
    io::Error::new(kind, error)
}
