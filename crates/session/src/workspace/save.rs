// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::{history::DocumentSnapshotTarget, identity::DocumentSaveError};
use std::{error::Error, fmt};

/// Save failure before adoption of the new workspace baseline.
#[derive(Debug)]
pub enum WorkspaceSaveError<F> {
    Veto { document: u64, reason: String },
    Workspace(WorkspaceError),
    OpenTransaction(u64),
    Preparation(F),
    Save(DocumentSaveError<F>),
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Persist one committed recipe, adopting identity and clean baseline only on success.
    /// Example: `workspace.save(id, "alpha", source, target)?;`.
    pub fn save<T: DocumentSnapshotTarget>(
        &mut self,
        id: u64,
        version: &str,
        source: &mut dyn GuidSource,
        target: &mut T,
    ) -> Result<(), WorkspaceSaveError<T::Failure>> {
        let snapshot = self.save_snapshot(id)?;
        self.observer
            .before_save(id, &snapshot)
            .map_err(|reason| WorkspaceSaveError::Veto {
                document: id,
                reason,
            })?;
        let digest = target
            .prepare(
                &snapshot,
                self.entries[&id].session.history.derived_high_water(),
            )
            .map_err(WorkspaceSaveError::Preparation)?;
        self.identities
            .save(id, digest, version, source, target)
            .map_err(WorkspaceSaveError::Save)?;
        let entry = self.entries.get_mut(&id).expect("validated document");
        // Save preparation requires a closed history; the synchronous save cannot open it.
        entry
            .session
            .history
            .mark_saved()
            .expect("validated closed history");
        entry.baseline = snapshot;
        entry.baseline_derived_high_water = entry.session.history.derived_high_water();
        entry.identity_dirty = false;
        self.observer.saved(id);
        Ok(())
    }

    pub(super) fn save_snapshot<F>(
        &self,
        id: u64,
    ) -> Result<DocumentSnapshot, WorkspaceSaveError<F>> {
        let session = self.session(id).map_err(WorkspaceSaveError::Workspace)?;
        if session.in_transaction() {
            return Err(WorkspaceSaveError::OpenTransaction(id));
        }
        Ok(session.snapshot().clone())
    }
}

impl<F: fmt::Display> fmt::Display for WorkspaceSaveError<F> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Veto { document, reason } => write!(
                output,
                "save document {document} veto {reason:?}: expected policy approval"
            ),
            Self::Workspace(error) => error.fmt(output),
            Self::OpenTransaction(id) => write!(
                output,
                "save document {id}: expected all bounded transactions closed"
            ),
            Self::Preparation(error) => write!(output, "prepare snapshot save: {error}"),
            Self::Save(error) => error.fmt(output),
        }
    }
}
impl<F: Error + 'static> Error for WorkspaceSaveError<F> {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Workspace(error) => Some(error),
            Self::Preparation(error) => Some(error),
            Self::Save(error) => Some(error),
            Self::OpenTransaction(_) | Self::Veto { .. } => None,
        }
    }
}
