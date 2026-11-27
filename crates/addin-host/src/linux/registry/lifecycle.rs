// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeAddInRegistry {
    /// Activate once unless disabled or stopping; example: `unsafe { host.activate(id) }?;`.
    /// # Safety
    /// Obey foreign affinity and keep owner dispatch available through activation and later cleanup.
    pub unsafe fn activate(&mut self, id: &str) -> Result<(), Error> {
        self.require_running()?;
        let index = self.entry_index(id)?;
        if self.load_behavior(id.into()) == LOAD_DISABLED {
            return Err(Error(format!(
                "add-in {id:?} is disabled: expected startup or demand policy"
            )));
        }
        let library = &mut self.entries[index].library;
        if library.state() == AddInState::Active {
            return Ok(());
        }
        let (call, free) = self.binding.callbacks();
        // SAFETY: callbacks remain owned by this host; the caller supplies dispatch and affinity.
        unsafe { library.activate(call, free) }
    }
    /// Stop one entry or retry quarantined cleanup; unknown IDs are a Go-compatible no-op.
    /// Usage: example: `unsafe { host.deactivate(id) }?;`.
    /// # Safety
    /// Keep dispatch available and use the required foreign thread until deactivation succeeds.
    pub unsafe fn deactivate(&mut self, id: &str) -> Result<(), Error> {
        let Some(index) = self.find_entry(id) else {
            return Ok(());
        };
        // SAFETY: owned live callbacks and caller-provided dispatch/affinity outlive cleanup.
        unsafe { self.entries[index].library.deactivate() }
    }
    /// Forward a notification only while running; example: `unsafe { host.notify(id, bytes) }?;`.
    /// # Safety
    /// Obey notify affinity and borrowed-payload rules; do not synchronously reenter a blocked owner.
    pub unsafe fn notify(&mut self, id: &str, payload: &[u8]) -> Result<(), Error> {
        self.require_running()?;
        let index = self.entry_index(id)?;
        // SAFETY: borrowed bytes stay live; caller enforces foreign affinity and no retention.
        unsafe { self.entries[index].library.notify(payload) }
    }
    /// Route the add-in's opaque automation contract; example: `unsafe { host.call_automation(id, method, json) }?;`.
    /// # Safety
    /// Obey automation affinity, no synchronous owner reentry, and trusted response allocator rules.
    pub unsafe fn call_automation(
        &mut self,
        id: &str,
        method: &str,
        payload: &[u8],
    ) -> Result<Vec<u8>, Error> {
        self.require_running()?;
        let index = self.entry_index(id)?;
        // SAFETY: canonical ABI and matching allocator; caller supplies the response trust/affinity.
        unsafe { self.entries[index].library.automation(method, payload) }
    }
    /// Stop admission and notifications, attempt every cleanup, then retire callbacks on success.
    /// Example: `unsafe { host.shutdown() }?;` before closing the owner queue.
    /// # Safety
    /// Drain dispatch concurrently where needed. Successful deactivation must join foreign callback workers.
    /// After failure retain this host and dispatch, then retry cleanup; do not assume foreign work stopped.
    pub unsafe fn shutdown(&mut self) -> Result<(), Vec<Error>> {
        if self.stage == RegistryStage::Closed {
            return Ok(());
        }
        self.stage = RegistryStage::Stopping;
        let mut failures = Vec::new();
        for entry in &mut self.entries {
            // SAFETY: caller keeps dispatch live and supplies every library's required affinity.
            if let Err(error) = unsafe { entry.library.deactivate() } {
                failures.push(error);
            }
        }
        if !failures.is_empty() {
            return Err(failures);
        }
        self.binding.close();
        self.stage = RegistryStage::Closed;
        Ok(())
    }
}
