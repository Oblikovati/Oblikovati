// SPDX-License-Identifier: GPL-2.0-only
//! An injected store for applied ribbon settings.

/// Stores the versioned ribbon settings owned by the desktop host.
/// Implement this interface to keep settings across GUI sessions.
///
/// ```ignore
/// let store: Box<dyn GuiRibbonSettingsStore> = desktop_settings();
/// engine.set_ribbon_settings_store(store)?;
/// ```
pub trait GuiRibbonSettingsStore {
    /// Read the last saved settings, or `None` when no settings exist.
    /// Example: `let saved = store.load()?;`.
    fn load(&mut self) -> Result<Option<Vec<u8>>, String>;
    /// Atomically save complete settings before they are applied.
    /// Example: `store.save(xml_bytes)?;`.
    fn save(&mut self, bytes: &[u8]) -> Result<(), String>;
}
