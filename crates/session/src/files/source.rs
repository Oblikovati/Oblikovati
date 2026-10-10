// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Resolve an open parameter source without I/O or activation; ambiguous names are errors.
    /// Example: `host.resolve_parameter_source("/parts/gear.opd")?`.
    pub fn resolve_parameter_source(&self, name: &str) -> Result<Option<u64>, FileWorkspaceError> {
        if name.is_empty() {
            return Ok(None);
        }
        let mut matching = self
            .retained
            .iter()
            .filter(|(_, file)| source_matches(file, name));
        let Some((&id, file)) = matching.next() else {
            return Ok(None);
        };
        if matching.next().is_some() {
            return Err(FileWorkspaceError::AmbiguousSource(name.into()));
        }
        let kind = DocumentType(file.package.manifest().document_type);
        Ok(matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY).then_some(id))
    }
}

fn source_matches(file: &RetainedFile, name: &str) -> bool {
    match &file.path {
        Some(path) => path == Path::new(name),
        None => file.display_override.as_deref() == Some(name),
    }
}
