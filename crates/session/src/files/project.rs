// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{DocumentType, NewFileMetadata};
use oblikovati_persistence::project::{ProjectFileLookup, ProjectFileProbe};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles + ProjectFileProbe>
    FileWorkspace<O, W, F>
{
    /// Create from the explicit metadata template or the project's template for the requested kind.
    /// No built-in content is substituted when a configured template is absent or invalid.
    /// Example: `host.create_from_project_template(project, DOCUMENT_PART, &metadata, observer, "alpha", source)?;`.
    pub fn create_from_project_template(
        &mut self,
        project: &ProjectFileLookup,
        kind: DocumentType,
        metadata: &NewFileMetadata,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        create::require_document_kind(kind.0)?;
        let template = self.select_project_template(project, kind, metadata)?;
        let (destination, package) = self.prepare_template(metadata, &template)?;
        let actual = package.manifest().document_type;
        if actual != kind.0 {
            return Err(FileWorkspaceError::TemplateKind {
                expected: kind.0,
                actual,
            });
        }
        self.create(&destination, package, observer, version, source)
    }

    fn select_project_template(
        &mut self,
        project: &ProjectFileLookup,
        kind: DocumentType,
        metadata: &NewFileMetadata,
    ) -> Result<PathBuf, FileWorkspaceError> {
        if !metadata.template_file.is_empty() {
            return Ok(PathBuf::from(&metadata.template_file));
        }
        project
            .template_file(kind, &mut self.files)
            .map_err(FileWorkspaceError::Project)?
            .ok_or(FileWorkspaceError::MissingTemplate(kind.0))
    }
}
