// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::NewFileMetadata;
use oblikovati_model::identity::FileIdentity;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Write an independent copy; the original path, identity and dirty state do not change.
    /// Example: `let copy_identity = host.save_copy(id, target_path, None, "alpha", source)?;`.
    pub fn save_copy(
        &mut self,
        id: u64,
        requested: &Path,
        metadata: Option<&NewFileMetadata>,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<FileIdentity, FileWorkspaceError> {
        self.retained_file(id)?;
        let destination = self.unowned_destination(requested)?;
        let original = &self.retained[&id];
        let mut package = copy_package(original, metadata);
        let mut target = ParameterPackageTarget::new(&mut package, &destination, &mut self.files);
        self.workspace
            .save_copy(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)
    }
}

fn copy_package(file: &RetainedFile, metadata: Option<&NewFileMetadata>) -> DocumentPackage {
    let name = name::saved_name(file, file.path.as_deref().unwrap_or(Path::new("")));
    let package = file.package.with_display_name(&name);
    metadata.map_or_else(
        || package.clone(),
        |metadata| package.with_copy_metadata(metadata),
    )
}
