// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Caller;
use std::sync::{
    Arc, Mutex,
    atomic::{AtomicUsize, Ordering},
};
mod buffers;
mod entry;
pub use buffers::HostBufferCounts;
struct CallbackEndpoint {
    transport: Arc<dyn Caller + Send + Sync>,
    active: AtomicUsize,
}
struct CallbackSlot {
    endpoint: Option<Arc<CallbackEndpoint>>,
    used: bool,
}
// The context-free C ABI needs one endpoint. It contains transport only, never model state.
static CALLBACK_SLOT: Mutex<CallbackSlot> = Mutex::new(CallbackSlot {
    endpoint: None,
    used: false,
});
/// One process-lifetime callback identity; retirement can never retarget stale native function pointers.
pub struct HostCallbackBinding {
    endpoint: Arc<CallbackEndpoint>,
}
impl HostCallbackBinding {
    /// Attach the queued transport before activation; its model remains on the drain owner.
    /// Example: `HostCallbackBinding::install(Arc::new(queued_caller))?;`.
    pub fn install(transport: Arc<dyn Caller + Send + Sync>) -> Result<Self, Error> {
        let mut slot = CALLBACK_SLOT.lock().map_err(|_| {
            Error("callback registry poisoned: expected an idle healthy endpoint".into())
        })?;
        if slot.used {
            return Err(Error(
                "callback identity already used: expected the first host binding in this process"
                    .into(),
            ));
        }
        let endpoint = Arc::new(CallbackEndpoint {
            transport,
            active: AtomicUsize::new(0),
        });
        slot.used = true;
        slot.endpoint = Some(Arc::clone(&endpoint));
        Ok(Self { endpoint })
    }
    /// Obtain exact API function pointers; example: `let (call, free) = binding.callbacks();`.
    /// Native callers must supply valid readable inputs/writable outputs and free each live returned buffer once.
    pub fn callbacks(&self) -> (ffi::ObkHostCall, ffi::ObkHostFree) {
        (entry::host_call, buffers::host_free)
    }
    /// Stop new admission after add-ins deactivate; already leased callbacks may finish.
    /// Example: `binding.close(); owner.close();`.
    pub fn close(&self) {
        retire_endpoint();
    }
    /// Inspect calls still using the retired endpoint; example: `binding.in_flight()` before joining workers.
    pub fn in_flight(&self) -> usize {
        self.endpoint.active.load(Ordering::Acquire)
    }
    /// Inspect owned response allocations and refused frees; example: `binding.buffer_counts()`.
    pub fn buffer_counts(&self) -> HostBufferCounts {
        buffers::counts()
    }
}
impl Drop for HostCallbackBinding {
    fn drop(&mut self) {
        self.close();
    }
}
struct CallbackLease(Arc<CallbackEndpoint>);
impl Drop for CallbackLease {
    fn drop(&mut self) {
        self.0.active.fetch_sub(1, Ordering::AcqRel);
    }
}
fn lease_endpoint() -> Result<CallbackLease, Error> {
    let slot = CALLBACK_SLOT.lock().map_err(|_| {
        Error("callback registry poisoned: expected a healthy registered transport".into())
    })?;
    let endpoint = slot.endpoint.as_ref().ok_or_else(|| {
        Error("host callbacks retired or not initialized: expected a live dispatch endpoint".into())
    })?;
    endpoint.active.fetch_add(1, Ordering::AcqRel);
    Ok(CallbackLease(Arc::clone(endpoint)))
}
fn retire_endpoint() {
    // Retirement is teardown, so it must stop admission even after registry poisoning.
    CALLBACK_SLOT
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .endpoint = None;
}
