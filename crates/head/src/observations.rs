// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_automation::DocumentObserverFactory;
use oblikovati_model::{history::DocumentSnapshot, identity::IdentityRegistration};
use oblikovati_session::{TransactionObserver, workspace::WorkspaceObserver};
use std::{
    cell::{Cell, RefCell},
    collections::BTreeMap,
    path::Path,
    rc::Rc,
};
/// Retained client warning for a document; this is desktop state, not a wire DTO.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MissingInterestNotice {
    pub document: u64,
    pub display_name: String,
    pub clients: Vec<String>,
}
#[derive(Default)]
struct ObservationState {
    changed: Cell<bool>,
    missing: RefCell<BTreeMap<u64, MissingInterestNotice>>,
}
/// Coalesced invalidation plus retained warnings; observers never capture the live document owner.
#[derive(Clone, Default)]
pub struct DesktopObservations {
    state: Rc<ObservationState>,
}
impl DesktopObservations {
    /// Consume invalidation without discarding warnings; example: `observations.take_changed()` before querying UI state.
    pub fn take_changed(&self) -> bool {
        self.state.changed.replace(false)
    }
    /// Copy current unavailable-client warnings in document ID order; example: `observations.missing_interests()` for a panel.
    pub fn missing_interests(&self) -> Vec<MissingInterestNotice> {
        self.state.missing.borrow().values().cloned().collect()
    }
    fn invalidate(&self) {
        self.state.changed.set(true);
    }
}
impl crate::HeadInvalidation for DesktopObservations {
    fn take_changed(&self) -> bool {
        DesktopObservations::take_changed(self)
    }
}
impl DocumentObserverFactory for DesktopObservations {
    type Observer = Self;
    fn create(&mut self) -> Result<Self, String> {
        Ok(self.clone())
    }
}
impl TransactionObserver for DesktopObservations {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {
        self.invalidate();
    }
    fn changed(&mut self, _: u64, _: u64) {
        self.invalidate();
    }
}
impl WorkspaceObserver for DesktopObservations {
    fn interest_client_present(&self, _: &str) -> bool {
        // No foreign registry is attached yet. Preserve missing-client diagnostics until it is bound.
        false
    }
    fn unavailable_interest_clients(
        &mut self,
        document: u64,
        display_name: &str,
        clients: &[String],
    ) {
        self.state.missing.borrow_mut().insert(
            document,
            MissingInterestNotice {
                document,
                display_name: display_name.into(),
                clients: clients.into(),
            },
        );
        self.invalidate();
    }
    fn created(&mut self, _: u64, _: IdentityRegistration) {
        self.invalidate();
    }
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {
        self.invalidate();
    }
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn saved(&mut self, _: u64) {
        self.invalidate();
    }
    fn opened(&mut self, _: u64, _: IdentityRegistration) {
        self.invalidate();
    }
    fn activated(&mut self, _: u64) {
        self.invalidate();
    }
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn closed(&mut self, document: u64, _: Option<u64>) {
        self.state.missing.borrow_mut().remove(&document);
        self.invalidate();
    }
}
#[cfg(test)]
#[path = "../tests/observations.rs"]
mod tests;
