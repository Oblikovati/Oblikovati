// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::history::DocumentSnapshotTarget;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Copy current committed recipe state without changing the source's saved baseline.
    /// Example: `let identity = workspace.save_copy(id, "alpha", source, target)?;`.
    pub fn save_copy<T: DocumentSnapshotTarget>(
        &self,
        id: u64,
        version: &str,
        source: &mut dyn GuidSource,
        target: &mut T,
    ) -> Result<FileIdentity, WorkspaceSaveError<T::Failure>> {
        let snapshot = self.save_snapshot(id)?;
        let digest = target
            .prepare(
                &snapshot,
                self.entries[&id].session.history.derived_high_water(),
                self.entries[&id].session.high_water(),
                self.entries[&id].session.history.sketch_reservations(),
                self.entries[&id].session.history.feature_high_water(),
            )
            .map_err(WorkspaceSaveError::Preparation)?;
        self.identities
            .save_copy(id, digest, version, source, target)
            .map_err(WorkspaceSaveError::Save)
    }
}
