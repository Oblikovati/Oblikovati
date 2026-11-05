// SPDX-License-Identifier: GPL-2.0-only
use super::ThemeDefinition;

/// The application saves a selected theme before publishing a new live palette.
pub trait ThemeSelectionStore {
    /// Persist the selection atomically; example: `store.save_selection(library.active())?`.
    fn save_selection(&mut self, theme: &ThemeDefinition) -> Result<(), String>;
}
