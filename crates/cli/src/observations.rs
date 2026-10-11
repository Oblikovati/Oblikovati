// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{history::DocumentSnapshot, identity::IdentityRegistration};
use oblikovati_session::{TransactionObserver, workspace::WorkspaceObserver};
use std::path::Path;

// A one-command headless process has no connected add-ins or event consumers.
pub(super) struct CliObservations;
impl TransactionObserver for CliObservations {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
impl WorkspaceObserver for CliObservations {
    fn interest_client_present(&self, _: &str) -> bool {
        false
    }
    fn unavailable_interest_clients(&mut self, _: u64, _: &str, _: &[String]) {}
    fn created(&mut self, _: u64, _: IdentityRegistration) {}
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {}
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn saved(&mut self, _: u64) {}
    fn opened(&mut self, _: u64, _: IdentityRegistration) {}
    fn activated(&mut self, _: u64) {}
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {}
}
