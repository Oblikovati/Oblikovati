// SPDX-License-Identifier: GPL-2.0-only

use super::*;

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
        self.path(id)?;
        let destination = checked_path(requested, self.files.save_path(requested))?;
        self.require_available_destination(id, &destination)?;
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
        if file.path == destination {
            return;
        }
        let previous = std::mem::replace(&mut file.path, destination.clone());
        self.paths.remove(&previous);
        self.paths.insert(destination.clone(), id);
        self.refresh_display_name(id);
        self.workspace
            .notify_path_changed(id, &previous, &destination);
    }
}
