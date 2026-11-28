// SPDX-License-Identifier: GPL-2.0-only
//! Silent transaction and workspace observers for the browser web session.

use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{history::DocumentSnapshot, identity::IdentityRegistration};
use oblikovati_session::{TransactionObserver, workspace::WorkspaceObserver};
use std::path::Path;

#[derive(Default)]
pub(super) struct SilentTransactionObserver;

impl TransactionObserver for SilentTransactionObserver {
    fn before(
        &mut self,
        _event: &TransactionEventPayload,
        _snapshot: &DocumentSnapshot,
    ) -> Result<(), String> {
        Ok(())
    }

    fn after(&mut self, _event: &TransactionEventPayload, _snapshot: &DocumentSnapshot) {}

    fn changed(&mut self, _document: u64, _generation: u64) {}
}

#[derive(Default)]
pub(super) struct SilentWorkspaceObserver;

impl WorkspaceObserver for SilentWorkspaceObserver {
    fn interest_client_present(&self, _client: &str) -> bool {
        false
    }

    fn unavailable_interest_clients(
        &mut self,
        _document: u64,
        _display_name: &str,
        _clients: &[String],
    ) {
    }

    fn created(&mut self, _document: u64, _registration: IdentityRegistration) {}

    fn path_changed(&mut self, _document: u64, _previous: &Path, _current: &Path) {}

    fn before_open(&mut self, _path: &Path) -> Result<(), String> {
        Ok(())
    }

    fn before_save(&mut self, _document: u64, _snapshot: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }

    fn saved(&mut self, _document: u64) {}

    fn opened(&mut self, _document: u64, _registration: IdentityRegistration) {}

    fn activated(&mut self, _document: u64) {}

    fn before_close(&mut self, _document: u64, _snapshot: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }

    fn closed(&mut self, _document: u64, _active: Option<u64>) {}
}
