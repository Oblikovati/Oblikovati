// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::{SaveDocumentAsArgs, SaveDocumentResult};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Save to a new bound path without deleting the old file; failed saves keep the old binding.
    /// Example: `host.save_as(id, new_path, "alpha", source)?;`.
    pub fn save_as(
        &mut self,
        id: u64,
        requested: &Path,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<(), FileWorkspaceError> {
        let destination = self.prepare_save_as(id, requested)?;
        self.save_as_to(id, destination, version, source)
    }

    /// Save As with a canonical response validated before storage changes.
    /// Example: `host.save_as_report(request, "alpha", source)?` for an API call.
    pub fn save_as_report(
        &mut self,
        request: SaveDocumentAsArgs,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<SaveDocumentResult, FileWorkspaceError> {
        let destination =
            self.prepare_save_as(request.document, Path::new(&request.new_full_document_name))?;
        let full_document_name = destination
            .to_str()
            .ok_or_else(|| FileWorkspaceError::NonUtf8ResponsePath(destination.clone()))?
            .into();
        self.save_as_to(request.document, destination, version, source)?;
        Ok(SaveDocumentResult { full_document_name })
    }

    fn prepare_save_as(
        &mut self,
        id: u64,
        requested: &Path,
    ) -> Result<PathBuf, FileWorkspaceError> {
        self.retained_file(id)?;
        let destination = checked_path(requested, self.files.save_path(requested))?;
        self.require_available_destination(id, &destination)?;
        Ok(destination)
    }

    fn save_as_to(
        &mut self,
        id: u64,
        destination: PathBuf,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<(), FileWorkspaceError> {
        let file = self.retained.get_mut(&id).expect("validated binding");
        let mut target =
            ParameterPackageTarget::new(&mut file.package, &destination, &mut self.files);
        self.workspace
            .save(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)?;
        self.rebind_path(id, destination);
        Ok(())
    }

    fn require_available_destination(
        &self,
        id: u64,
        path: &Path,
    ) -> Result<(), FileWorkspaceError> {
        if let Some(&owner) = self.paths.get(path)
            && owner != id
        {
            return Err(FileWorkspaceError::DestinationInUse {
                path: path.into(),
                owner,
            });
        }
        Ok(())
    }

    fn rebind_path(&mut self, id: u64, destination: PathBuf) {
        let file = self.retained.get_mut(&id).expect("validated binding");
        if file.path.as_ref() == Some(&destination) {
            return;
        }
        let previous = file.path.replace(destination.clone());
        if let Some(path) = &previous {
            self.paths.remove(path);
        }
        self.paths.insert(destination.clone(), id);
        self.refresh_display_name(id);
        // The observer contract uses an empty prior path for the first storage binding.
        self.workspace.notify_path_changed(
            id,
            previous.as_deref().unwrap_or_else(|| Path::new("")),
            &destination,
        );
    }
}
