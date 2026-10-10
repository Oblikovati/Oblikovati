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
        let original = &self.retained[&id].package;
        let mut package = metadata.map_or_else(
            || original.clone(),
            |metadata| original.with_copy_metadata(metadata),
        );
        let mut target = ParameterPackageTarget::new(&mut package, &destination, &mut self.files);
        self.workspace
            .save_copy(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)
    }
}
