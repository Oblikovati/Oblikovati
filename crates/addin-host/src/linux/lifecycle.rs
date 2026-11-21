// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{cell::Cell, marker::PhantomData};
/// Failed lifecycle calls permit cleanup only; partial foreign initialization cannot be rolled back by Rust.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AddInState {
    Inactive,
    Active,
    NeedsCleanup,
}
/// One lifecycle wrapper; !Sync prevents sharing it, not isolation from other library opens.
pub struct LoadedAddIn {
    exports: exports::ProbeExports,
    metadata: AddInProbe,
    state: AddInState,
    exclusive: PhantomData<Cell<()>>,
}
impl LoadedAddIn {
    pub(super) fn new(exports: exports::ProbeExports, metadata: AddInProbe) -> Self {
        Self {
            exports,
            metadata,
            state: AddInState::Inactive,
            exclusive: PhantomData,
        }
    }
    /// Read the copied admission evidence; example: `addin.metadata().id`.
    pub fn metadata(&self) -> &AddInProbe {
        &self.metadata
    }
    /// Read lifecycle status; example: `addin.state() == AddInState::Active`.
    pub fn state(&self) -> AddInState {
        self.state
    }
    /// Activate once with live host callbacks; failure requires explicit cleanup.
    /// Example: `unsafe { addin.activate(host_call, host_free) }?;`.
    /// # Safety
    /// Use the library's required thread. Callbacks and their context must remain valid until successful deactivation.
    pub unsafe fn activate(
        &mut self,
        call: ffi::ObkHostCall,
        free: ffi::ObkHostFree,
    ) -> Result<(), Error> {
        self.require_state(AddInState::Inactive, "ObkAddInActivate")?;
        self.state = AddInState::NeedsCleanup;
        // SAFETY: trusted export and caller-provided live callbacks obey the published ABI.
        let status = unsafe { (self.exports.activate)(Some(call), Some(free)) };
        self.check_status("ObkAddInActivate", status)?;
        self.state = AddInState::Active;
        Ok(())
    }
    /// Deactivate or clean up an unsuccessful lifecycle attempt; already inactive is a no-op.
    /// Example: `unsafe { addin.deactivate() }?;`.
    /// # Safety
    /// Use the library's required thread; keep callback dispatch available until this returns success.
    pub unsafe fn deactivate(&mut self) -> Result<(), Error> {
        if self.state == AddInState::Inactive {
            return Ok(());
        }
        self.state = AddInState::NeedsCleanup;
        // SAFETY: canonical export in a resident mapping, invoked with the caller's affinity guarantee.
        let status = unsafe { (self.exports.deactivate)() };
        self.check_status("ObkAddInDeactivate", status)?;
        self.state = AddInState::Inactive;
        Ok(())
    }
    /// Notify an active library synchronously; empty payloads use null plus zero.
    /// Example: `unsafe { addin.notify(&event_json) }?;`.
    /// # Safety
    /// Obey foreign affinity; the library must not retain the borrowed event bytes after return.
    pub unsafe fn notify(&mut self, event: &[u8]) -> Result<(), Error> {
        self.require_state(AddInState::Active, "ObkAddInNotify")?;
        let length = response::c_length(event.len())?;
        // SAFETY: slice lives through the call; canonical length is checked before conversion.
        let status = unsafe { (self.exports.notify)(response::payload_pointer(event), length) };
        self.check_status("ObkAddInNotify", status)
    }
    pub(super) fn require_state(&self, expected: AddInState, entry: &str) -> Result<(), Error> {
        if self.state != expected {
            return Err(Error(format!(
                "add-in {:?} {entry} in state {:?}: expected {expected:?}",
                self.metadata.id, self.state
            )));
        }
        Ok(())
    }
    pub(super) fn check_status(&self, entry: &str, status: i32) -> Result<(), Error> {
        if status != ffi::OBK_OK {
            return Err(Error(format!(
                "add-in {:?} {entry} returned {status}: expected OBK_OK ({})",
                self.metadata.id,
                ffi::OBK_OK
            )));
        }
        Ok(())
    }
}
// Drop performs no foreign work: deactivation can need owner dispatch and must be orchestrated explicitly.
impl LoadedAddIn {
    /// Copy and free the optional automation reply, including error buffers, through ObkFree.
    /// Example: `let reply = unsafe { addin.automation("method", &json) }?;`.
    /// # Safety
    /// Obey affinity. Any nonnull reply must be valid for ObkFree; nonnull positive-length replies
    /// must provide that many readable bytes. Invalid signed lengths and null combinations are declined.
    pub unsafe fn automation(&mut self, method: &str, arguments: &[u8]) -> Result<Vec<u8>, Error> {
        self.require_state(AddInState::Active, "ObkAddInAutomation")?;
        let export = self.exports.automation.ok_or_else(|| {
            Error(format!(
                "add-in {:?}: expected optional ObkAddInAutomation export",
                self.metadata.id
            ))
        })?;
        let name = CString::new(method).map_err(|error| {
            Error(format!(
                "automation method {method:?}: expected no embedded NUL; {error}"
            ))
        })?;
        let length = response::c_length(arguments.len())?;
        // SAFETY: canonical export, live borrowed request and trusted add-in allocator/free pair.
        unsafe { response::invoke(export, self.exports.free, &name, arguments, length) }
    }
}
