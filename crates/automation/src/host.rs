// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    EDIT_METHODS, QUERY_METHODS, SessionQueryHost, TRANSACTION_METHODS, edit::PreparedEdit,
    transactions,
};
use oblikovati_api::{Error, client::Caller};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
use std::cell::RefCell;

/// Synchronous transport used on the session owner, before a native/queued transport is attached.
pub struct SessionHost<'a, O, W> {
    workspace: RefCell<&'a mut WorkspaceSession<O, W>>,
}

impl<'a, O: TransactionObserver, W: WorkspaceObserver> SessionHost<'a, O, W> {
    /// Borrow exclusive command access; example: `Client::new(SessionHost::new(&mut workspace))`.
    pub fn new(workspace: &'a mut WorkspaceSession<O, W>) -> Self {
        Self {
            workspace: RefCell::new(workspace),
        }
    }

    fn edit(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut workspace = self.workspace.try_borrow_mut().map_err(|_| busy(method))?;
        let id = workspace
            .active_document()
            .ok_or_else(|| Error(format!("method {method:?}: expected an active document")))?;
        let edit = PreparedEdit::decode(method, request)?;
        workspace
            .commands(id)
            .map_err(|error| Error(error.to_string()))?
            .run(edit.label(), |snapshot, reservations| {
                // Encode before committing, so a response failure cannot leave an unreported edit.
                edit.apply(&mut snapshot.parameters, reservations)
            })
            .map_err(|error| Error(error.to_string()))
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> Caller for SessionHost<'_, O, W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if QUERY_METHODS.contains(&method) {
            let workspace = self.workspace.try_borrow().map_err(|_| busy(method))?;
            return SessionQueryHost::new(&workspace).call(method, request);
        }
        if TRANSACTION_METHODS.contains(&method) {
            let mut workspace = self.workspace.try_borrow_mut().map_err(|_| busy(method))?;
            return transactions::apply(&mut workspace, method, request);
        }
        if EDIT_METHODS.contains(&method) {
            return self.edit(method, request);
        }
        Err(Error(format!(
            "method {method:?}: expected a supported query from {QUERY_METHODS:?} or edit from {EDIT_METHODS:?} or transaction from {TRANSACTION_METHODS:?}"
        )))
    }
}

fn busy(method: &str) -> Error {
    Error(format!(
        "method {method:?}: expected an idle session owner; synchronous reentry is refused"
    ))
}

#[cfg(test)]
#[path = "host_tests.rs"]
mod tests;
