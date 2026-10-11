// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    contract::AddInRegistry,
    types::{AddInLoadBehavior, LOAD_ON_STARTUP},
};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
pub(super) struct CatalogCaller<'a, C> {
    pub catalog: &'a [AddInInfo],
    pub model: &'a C,
}
impl<C: Caller> Caller for CatalogCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_ADD_INS_LIST => encode_wire_reply(&ListAddInsResult {
                add_ins: Some(self.catalog.to_vec()),
            }),
            METHOD_ADD_INS_GET => {
                let arguments: AddInRefArgs = decode_wire_arguments(method, request)?;
                encode_wire_reply(&describe(self.catalog, &arguments.id)?)
            }
            _ => self.model.call(method, request),
        }
    }
}
pub(super) fn describe(catalog: &[AddInInfo], id: &str) -> Result<AddInInfo, Error> {
    catalog
        .iter()
        .find(|entry| entry.id == id)
        .cloned()
        .ok_or_else(|| Error(format!("add-in {id:?}: expected a registered ID")))
}
impl<W: DispatchWakeup + 'static> NativeAddInRuntime<W> {
    /// Read the last completed registration snapshot; example: `runtime.describe(id)?;`.
    pub fn describe(&self, id: &str) -> Result<AddInInfo, Error> {
        describe(&self.catalog, id)
    }
    /// Inspect cleanup quarantine after completion; example: `runtime.lifecycle_state(id)?;`.
    pub fn lifecycle_state(&self, id: &str) -> Result<AddInState, Error> {
        self.states
            .iter()
            .find(|(key, _)| key == id)
            .map(|(_, state)| *state)
            .ok_or_else(|| Error(format!("add-in {id:?}: expected a registered lifecycle")))
    }
    /// Read copied allocation counts at the last native completion, not an asynchronous census.
    /// Example: `runtime.last_buffer_counts().outstanding` after successful shutdown.
    pub fn last_buffer_counts(&self) -> HostBufferCounts {
        self.counts
    }
}
impl<W: DispatchWakeup + 'static> AddInRegistry for NativeAddInRuntime<W> {
    fn is_active(&self, id: String) -> bool {
        self.catalog
            .iter()
            .any(|entry| entry.id == id && entry.activated)
    }
    fn load_behavior(&self, id: String) -> AddInLoadBehavior {
        self.catalog
            .iter()
            .find(|entry| entry.id == id)
            .map(|entry| entry.load_behavior)
            .unwrap_or(LOAD_ON_STARTUP)
    }
    fn registered(&self) -> Option<Vec<String>> {
        Some(self.catalog.iter().map(|entry| entry.id.clone()).collect())
    }
}
