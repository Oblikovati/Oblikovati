// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::history::DocumentSnapshotTarget;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Activate an open document; example: `workspace.activate(id)?;`.
    pub fn activate(&mut self, id: u64) -> Result<(), WorkspaceError> {
        self.session(id)?;
        if self.active != Some(id) {
            self.focus_identity = std::sync::Arc::new(());
        }
        self.active = Some(id);
        self.observer.activated(id);
        Ok(())
    }

    /// Explicitly discard a document after close policy accepts; no save is attempted.
    /// Example: `let observer = workspace.close_discarding_changes(id)?;`.
    pub fn close_discarding_changes(&mut self, id: u64) -> Result<O, WorkspaceError> {
        self.approve_close(id)?;
        self.remove_closed_document(id)
    }

    /// Save a dirty document before close; veto or failure leaves it open.
    /// Example: `workspace.close_saving_changes(id, "alpha", source, target)?;`.
    pub fn close_saving_changes<T: DocumentSnapshotTarget>(
        &mut self,
        id: u64,
        version: &str,
        source: &mut dyn GuidSource,
        target: &mut T,
    ) -> Result<O, WorkspaceSaveError<T::Failure>> {
        if self
            .session(id)
            .map_err(WorkspaceSaveError::Workspace)?
            .in_transaction()
        {
            return Err(WorkspaceSaveError::OpenTransaction(id));
        }
        self.approve_close(id)
            .map_err(WorkspaceSaveError::Workspace)?;
        if self.needs_save(id).map_err(WorkspaceSaveError::Workspace)? {
            self.save(id, version, source, target)?;
        }
        self.remove_closed_document(id)
            .map_err(WorkspaceSaveError::Workspace)
    }

    fn approve_close(&mut self, id: u64) -> Result<(), WorkspaceError> {
        let entry = self.entries.get(&id).ok_or(WorkspaceError::Missing(id))?;
        self.observer
            .before_close(id, entry.session.snapshot())
            .map_err(|reason| WorkspaceError::CloseVeto {
                document: id,
                reason,
            })
    }

    fn remove_closed_document(&mut self, id: u64) -> Result<O, WorkspaceError> {
        self.identities.close(id)?;
        let entry = self.entries.remove(&id).expect("validated document");
        self.order.retain(|candidate| *candidate != id);
        if self.active == Some(id) {
            self.focus_identity = std::sync::Arc::new(());
            self.active = self.order.first().copied();
        }
        let observer = entry.session.close();
        self.observer.closed(id, self.active);
        Ok(observer)
    }
}
