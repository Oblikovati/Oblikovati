// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::DocumentSnapshot, identity::IdentityRegistration};
use oblikovati_session::{TransactionObserver, workspace::WorkspaceObserver};
#[derive(Default)]
pub(super) struct WorkflowObserver {
    events: Vec<String>,
}
impl TransactionObserver for WorkflowObserver {
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        _: &DocumentSnapshot,
    ) -> Result<(), String> {
        self.events.push(event.type_.clone());
        Ok(())
    }
    fn after(&mut self, event: &TransactionEventPayload, _: &DocumentSnapshot) {
        self.events.push(event.type_.clone());
    }
    fn changed(&mut self, _: u64, _: u64) {
        self.events.push("generation".into());
    }
}
impl WorkspaceObserver for WorkflowObserver {
    fn interest_client_present(&self, _: &str) -> bool {
        false
    }
    fn unavailable_interest_clients(&mut self, _: u64, _: &str, _: &[String]) {
        self.events.push("missing clients".into());
    }
    fn created(&mut self, _: u64, _: IdentityRegistration) {
        self.events.push("created".into());
    }
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {
        self.events.push("path changed".into());
    }
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn saved(&mut self, _: u64) {
        self.events.push("saved".into());
    }
    fn opened(&mut self, _: u64, _: IdentityRegistration) {
        self.events.push("opened".into());
    }
    fn activated(&mut self, _: u64) {
        self.events.push("activated".into());
    }
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {
        self.events.push("closed".into());
    }
}
pub(super) struct WorkflowFactory;
impl DocumentObserverFactory for WorkflowFactory {
    type Observer = WorkflowObserver;
    fn create(&mut self) -> Result<Self::Observer, String> {
        Ok(WorkflowObserver::default())
    }
}
