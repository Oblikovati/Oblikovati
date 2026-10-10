// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::NewFileMetadata;
use oblikovati_persistence::load::read_package;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Seed an unsaved independent document from an explicit template, preserving the template file.
    /// Empty display-name/subtype fields inherit from the template; both paths are required.
    /// Example: `host.create_from_template(&metadata, observer, "alpha", source)?;`.
    pub fn create_from_template(
        &mut self,
        metadata: &NewFileMetadata,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        let template = required_metadata_path("templateFile", &metadata.template_file)?;
        let (destination, package) = self.prepare_template(metadata, template)?;
        self.create(&destination, package, observer, version, source)
    }

    pub(super) fn prepare_template(
        &mut self,
        metadata: &NewFileMetadata,
        template: &Path,
    ) -> Result<(PathBuf, DocumentPackage), FileWorkspaceError> {
        let requested = required_metadata_path("fileName", &metadata.file_name)?;
        let destination = self.unowned_destination(requested)?;
        let template = checked_path(template, self.files.canonical_path(template))?;
        if destination == template {
            return Err(FileWorkspaceError::TemplateDestination(destination));
        }
        let package = read_package(&template, &mut self.files)
            .map_err(FileWorkspaceError::Load)?
            .with_copy_metadata(metadata);
        Ok((destination, package))
    }
}

fn required_metadata_path<'a>(
    field: &'static str,
    value: &'a str,
) -> Result<&'a Path, FileWorkspaceError> {
    if value.is_empty() {
        return Err(FileWorkspaceError::CreationMetadata {
            field,
            value: value.into(),
        });
    }
    Ok(Path::new(value))
}
