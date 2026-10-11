// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeAddInRegistry {
    /// Copy canonical identity and current runtime state; example: `host.describe(id)?;`.
    pub fn describe(&self, id: &str) -> Result<AddInInfo, Error> {
        let entry = &self.entries[self.entry_index(id)?];
        let mut identity = entry.identity.clone();
        identity.activated = entry.library.state() == AddInState::Active;
        identity.load_behavior = self.load_behavior(id.into());
        Ok(identity)
    }
    /// Read cleanup quarantine separately from the public active flag; example: `host.lifecycle_state(id)?;`.
    pub fn lifecycle_state(&self, id: &str) -> Result<AddInState, Error> {
        Ok(self.entries[self.entry_index(id)?].library.state())
    }
    /// Select startup entries in registration order; example: `for id in host.startup_ids() { ... }`.
    pub fn startup_ids(&self) -> Vec<String> {
        self.entries
            .iter()
            .filter(|entry| self.load_behavior(entry.identity.id.clone()) == LOAD_ON_STARTUP)
            .map(|entry| entry.identity.id.clone())
            .collect()
    }
}
impl AddInRegistry for NativeAddInRegistry {
    fn is_active(&self, id: String) -> bool {
        self.lifecycle_state(&id) == Ok(AddInState::Active)
    }
    fn load_behavior(&self, id: String) -> AddInLoadBehavior {
        self.behaviors.get(&id).copied().unwrap_or(LOAD_ON_STARTUP)
    }
    fn registered(&self) -> Option<Vec<String>> {
        Some(
            self.entries
                .iter()
                .map(|entry| entry.identity.id.clone())
                .collect(),
        )
    }
}
