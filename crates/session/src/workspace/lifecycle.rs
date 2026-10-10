// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Activate an open document; example: `workspace.activate(id)?;`.
    pub fn activate(&mut self, id: u64) -> Result<(), WorkspaceError> {
        self.session(id)?;
        self.active = Some(id);
        self.observer.activated(id);
        Ok(())
    }

    /// Explicitly discard a document after close policy accepts; no save is attempted.
    /// Example: `let observer = workspace.close_discarding_changes(id)?;`.
    pub fn close_discarding_changes(&mut self, id: u64) -> Result<O, WorkspaceError> {
        let entry = self.entries.get(&id).ok_or(WorkspaceError::Missing(id))?;
        self.observer
            .before_close(id, entry.session.snapshot())
            .map_err(|reason| WorkspaceError::CloseVeto {
                document: id,
                reason,
            })?;
        self.identities.close(id)?;
        let entry = self.entries.remove(&id).expect("validated document");
        self.order.retain(|candidate| *candidate != id);
        if self.active == Some(id) {
            self.active = self.order.first().copied();
        }
        let observer = entry.session.close();
        self.observer.closed(id, self.active);
        Ok(observer)
    }
}
