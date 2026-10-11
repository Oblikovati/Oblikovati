// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    EDIT_METHODS, QUERY_METHODS, SessionQueryHost, TRANSACTION_METHODS, command_owner, transactions,
};
use oblikovati_api::{Error, client::Caller};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
use std::cell::RefCell;
mod selection;

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
        command_owner::edit(&mut **workspace, method, request)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> SessionHost<'_, O, W> {
    fn edit_inference(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut workspace = self.workspace.try_borrow_mut().map_err(|_| busy(method))?;
        workspace
            .set_sketch_inference_options_projected(
                &crate::decode(method, request)?,
                crate::sketch_inference::inference_reply,
            )
            .map_err(|error| Error(error.to_string()))
    }
    fn query_owned_session(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let workspace = self.workspace.try_borrow().map_err(|_| busy(method))?;
        SessionQueryHost::new(&workspace).call(method, request)
    }
    fn command_owned_session(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut workspace = self.workspace.try_borrow_mut().map_err(|_| busy(method))?;
        if crate::CLIENT_COMMAND_METHODS.contains(&method) {
            return crate::clients::apply(workspace.connected_clients_mut(), method, request);
        }
        transactions::apply(&mut **workspace, method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver> Caller for SessionHost<'_, O, W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if method == oblikovati_api::wire::METHOD_SKETCH_SET_INFERENCE_OPTIONS {
            return self.edit_inference(method, request);
        }
        if QUERY_METHODS.contains(&method) {
            return self.query_owned_session(method, request);
        }
        if crate::CLIENT_COMMAND_METHODS.contains(&method) || TRANSACTION_METHODS.contains(&method)
        {
            return self.command_owned_session(method, request);
        }
        if EDIT_METHODS.contains(&method) {
            return self.edit(method, request);
        }
        Err(Error(format!(
            "method {method:?}: expected a supported query from {QUERY_METHODS:?} or edit from {EDIT_METHODS:?} or transaction from {TRANSACTION_METHODS:?} or sketch inference setter"
        )))
    }
}

pub(super) fn busy(method: &str) -> Error {
    Error(format!(
        "method {method:?}: expected an idle session owner; synchronous reentry is refused"
    ))
}

#[cfg(test)]
#[path = "host_tests.rs"]
pub(super) mod tests;
