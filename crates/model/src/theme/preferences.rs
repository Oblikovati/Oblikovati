// SPDX-License-Identifier: GPL-2.0-only
use super::ThemeDefinition;

/// The application saves a selected theme before publishing a new live palette.
pub trait ThemeSelectionStore {
    /// Persist the selection atomically; example: `store.save_selection(library.active())?`.
    fn save_selection(&mut self, theme: &ThemeDefinition) -> Result<(), String>;
    /// Report writable source identity before enabling Save; example: `store.can_save_custom(theme)`.
    fn can_save_custom(&self, _theme: &ThemeDefinition) -> bool {
        false
    }
    /// Save a known custom snapshot; example: `store.save_custom_theme(theme)?`.
    fn save_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        Err(format!(
            "theme {:?}: expected storage with a known custom XML source",
            theme.name()
        ))
    }
}
