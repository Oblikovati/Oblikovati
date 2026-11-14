// SPDX-License-Identifier: GPL-2.0-only
use super::{AddInState, HostCallbackBinding, LoadedAddIn, open_addin};
use oblikovati_api::{Error, contract::AddInRegistry, types::*, wire::*};
use std::{collections::BTreeMap, path::Path};
mod lifecycle;
mod queries;
struct RegisteredLibrary {
    library: LoadedAddIn,
    identity: AddInInfo,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum RegistryStage {
    Running,
    Stopping,
    Closed,
}
/// Ordered native registrations and explicit cleanup, with one owned callback identity.
/// Drop retires callbacks but does no foreign work; call shutdown while owner dispatch is live.
pub struct NativeAddInRegistry {
    entries: Vec<RegisteredLibrary>,
    behaviors: BTreeMap<String, AddInLoadBehavior>,
    binding: HostCallbackBinding,
    stage: RegistryStage,
}
impl NativeAddInRegistry {
    /// Own callbacks before loading any library; example: `NativeAddInRegistry::new(binding)`.
    pub fn new(binding: HostCallbackBinding) -> Self {
        Self {
            entries: Vec::new(),
            behaviors: BTreeMap::new(),
            binding,
            stage: RegistryStage::Running,
        }
    }
    /// Admit a trusted library once, without activating it; example: `unsafe { host.load(path) }?;`.
    /// # Safety
    /// Trust constructors, exports and readable metadata as required by open_addin.
    /// Do not open an alias of a registered library through a separate host.
    pub unsafe fn load(&mut self, path: &Path) -> Result<String, Error> {
        self.require_running()?;
        // SAFETY: caller supplies the same trust and exclusivity guarantees as open_addin.
        let library = unsafe { open_addin(path) }?;
        let id = library.metadata().id.clone();
        if id.is_empty() || self.entries.iter().any(|entry| entry.identity.id == id) {
            return Err(Error(format!(
                "add-in ID {id:?}: expected a nonempty unique registration"
            )));
        }
        let identity = library_identity(&library, path);
        self.entries.push(RegisteredLibrary { library, identity });
        Ok(id)
    }
    /// Remove only an inactive registration; example: `host.unregister("example")?;`.
    /// Its load preference survives a subsequent registration, as in the Go host.
    pub fn unregister(&mut self, id: &str) -> Result<(), Error> {
        let index = self.entry_index(id)?;
        let state = self.entries[index].library.state();
        if state != AddInState::Inactive {
            return Err(Error(format!(
                "add-in {id:?} state {state:?}: expected Inactive before unregister"
            )));
        }
        self.entries.remove(index);
        Ok(())
    }
    /// Set a known entry's policy without stopping it; example: `host.set_load_behavior(id, LOAD_DISABLED)?;`.
    pub fn set_load_behavior(
        &mut self,
        id: &str,
        behavior: AddInLoadBehavior,
    ) -> Result<(), Error> {
        self.require_running()?;
        self.entry_index(id)?;
        if ![LOAD_ON_STARTUP, LOAD_ON_DEMAND, LOAD_DISABLED].contains(&behavior) {
            return Err(Error(format!(
                "load behavior {} for {id:?}: expected startup, demand or disabled",
                behavior.0
            )));
        }
        self.behaviors.insert(id.into(), behavior);
        Ok(())
    }
    fn find_entry(&self, id: &str) -> Option<usize> {
        self.entries
            .iter()
            .position(|entry| entry.identity.id == id)
    }
    fn entry_index(&self, id: &str) -> Result<usize, Error> {
        self.find_entry(id)
            .ok_or_else(|| Error(format!("add-in {id:?}: expected a registered ID")))
    }
    fn require_running(&self) -> Result<(), Error> {
        if self.stage != RegistryStage::Running {
            return Err(Error(format!(
                "add-in registry state {:?}: expected Running",
                self.stage
            )));
        }
        Ok(())
    }
}
fn library_identity(library: &LoadedAddIn, path: &Path) -> AddInInfo {
    let manifest: AddInManifest =
        serde_json::from_str(&library.metadata().manifest).unwrap_or_default();
    AddInInfo {
        id: library.metadata().id.clone(),
        display_name: manifest.display_name,
        version: manifest.version,
        description: manifest.description,
        kind: manifest.kind,
        capabilities: manifest.capabilities,
        location: path.to_string_lossy().into_owned(),
        has_automation: library.metadata().has_automation,
        ..Default::default()
    }
}
