// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::AddInLoadBehavior;
impl<W: DispatchWakeup + 'static> NativeAddInRuntime<W> {
    /// Start trusted library loading; example: `unsafe { runtime.start_load(path) }?;`.
    /// # Safety
    /// Trust all constructors, exports and reply allocators. The library must support one assigned worker thread.
    /// Do not create another lifecycle alias for the same native library.
    pub unsafe fn start_load(&mut self, path: &Path) -> Result<(), Error> {
        self.begin(NativeAction::Load(path.into()))
    }
    /// Begin activation without blocking native frames; example: `unsafe { runtime.start_activate(id) }?;`.
    /// # Safety
    /// The admitted library must obey the canonical ABI and join callback work when deactivated.
    pub unsafe fn start_activate(&mut self, id: &str) -> Result<(), Error> {
        self.begin(NativeAction::Activate(id.into()))
    }
    /// Begin cleanup on the same foreign thread; example: `unsafe { runtime.start_deactivate(id) }?;`.
    /// # Safety
    /// Keep polling the owner queue; successful cleanup must mean foreign callback workers stopped.
    pub unsafe fn start_deactivate(&mut self, id: &str) -> Result<(), Error> {
        self.begin(NativeAction::Deactivate(id.into()))
    }
    /// Schedule policy changes in native control order; example: `runtime.start_policy(id, behavior)?;`.
    pub fn start_policy(&mut self, id: &str, behavior: AddInLoadBehavior) -> Result<(), Error> {
        self.begin(NativeAction::Policy(id.into(), behavior))
    }
    /// Copy an event into worker ownership; example: `unsafe { runtime.start_notify(id, bytes) }?;`.
    /// # Safety
    /// Trust the event export and no-retention contract, and preserve host event order.
    pub unsafe fn start_notify(&mut self, id: &str, payload: &[u8]) -> Result<(), Error> {
        self.begin(NativeAction::Notify(id.into(), payload.into()))
    }
    /// Copy an opaque request for the native automation export; example: `unsafe { runtime.start_automation(id, method, bytes) }?;`.
    /// # Safety
    /// Trust readable reply extents and matching ObkFree; the add-in must obey the no-synchronous-owner-reentry rule.
    pub unsafe fn start_automation(
        &mut self,
        id: &str,
        method: &str,
        payload: &[u8],
    ) -> Result<(), Error> {
        self.begin(NativeAction::Automation(
            id.into(),
            method.into(),
            payload.into(),
        ))
    }
    /// Begin all-entry cleanup; successful completion retires callbacks and joins the native worker.
    /// Example: `unsafe { runtime.start_shutdown() }?; runtime.wait(&host)?;` before dropping model ownership.
    /// # Safety
    /// Keep polling or waiting; each add-in must join foreign callback work before cleanup reports success.
    pub unsafe fn start_shutdown(&mut self) -> Result<(), Error> {
        self.begin(NativeAction::Shutdown)
    }
}
