// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    contract::AddInRegistry,
    types::{AddInLoadBehavior, LOAD_ON_STARTUP},
};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
/// A completed catalog snapshot over an injected model caller; it contains no native exports.
pub struct NativeCatalogCaller<C> {
    catalog: Vec<AddInInfo>,
    model: C,
}
impl<C: Caller> Caller for NativeCatalogCaller<C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        call_with_catalog(&self.catalog, &self.model, method, request)
    }
}
impl<C: Caller> Caller for &NativeCatalogCaller<C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        NativeCatalogCaller::call(self, method, request)
    }
}
pub(super) struct CatalogCaller<'a, C> {
    pub catalog: &'a [AddInInfo],
    pub model: &'a C,
}
impl<C: Caller> Caller for CatalogCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        call_with_catalog(self.catalog, self.model, method, request)
    }
}
fn call_with_catalog<C: Caller>(
    catalog: &[AddInInfo],
    model: &C,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_ADD_INS_LIST => encode_wire_reply(&ListAddInsResult {
            add_ins: Some(catalog.to_vec()),
        }),
        METHOD_ADD_INS_GET => {
            let arguments: AddInRefArgs = decode_wire_arguments(method, request)?;
            encode_wire_reply(&describe(catalog, &arguments.id)?)
        }
        _ => model.call(method, request),
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
    /// Copy the latest completed catalog for an owner-side caller; example: `Client::new(runtime.catalog_caller(host))`.
    pub fn catalog_caller<C: Caller>(&self, model: C) -> NativeCatalogCaller<C> {
        NativeCatalogCaller {
            catalog: self.catalog.clone(),
            model,
        }
    }
    /// Inspect an in-flight foreign control; example: `runtime.control_pending()` before close.
    pub fn control_pending(&self) -> bool {
        self.pending.is_some()
    }
    /// Inspect ready owner calls separately from a busy foreign worker; example: `runtime.queued_calls_pending()?` before sleeping.
    pub fn queued_calls_pending(&self) -> Result<bool, Error> {
        self.owner.pending().map(|count| count != 0)
    }
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

#[cfg(test)]
#[path = "../../../tests/runtime/catalog.rs"]
mod tests;
