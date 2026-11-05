// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::AtomicFiles, load::PackageFileReader, theme_files::save_custom_theme,
    theme_preferences::ThemePreferenceFile,
};
use oblikovati_model::theme::{ThemeDefinition, ThemeSelectionStore};
use std::{collections::BTreeMap, path::PathBuf};

/// Custom edits retain loaded filenames instead of deriving a second file from a display name.
pub struct ThemeFilePreferences<Files> {
    selection: ThemePreferenceFile<Files>,
    custom_files: Files,
    sources: BTreeMap<String, PathBuf>,
}
impl<Files: AtomicFiles + PackageFileReader> ThemeFilePreferences<Files> {
    /// Attach discovered source identity; example: `ThemeFilePreferences::new(preferences, files, loaded.sources)`.
    pub fn new(
        selection: ThemePreferenceFile<Files>,
        custom_files: Files,
        sources: BTreeMap<String, PathBuf>,
    ) -> Self {
        Self {
            selection,
            custom_files,
            sources,
        }
    }
    /// Read the loaded selection; example: `ThemeLibrary::new(customs, preferences.active_name())?`.
    pub fn active_name(&self) -> &str {
        self.selection.active_name()
    }
    /// Save to the loaded custom file atomically; example: `preferences.save_custom(&edited)?`.
    pub fn save_custom(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        let path = self.sources.get(theme.name()).ok_or_else(|| {
            format!(
                "custom theme {:?}: expected a known loaded XML source file",
                theme.name()
            )
        })?;
        save_custom_theme(path, theme, &mut self.custom_files)
    }
}
impl<Files: AtomicFiles + PackageFileReader> ThemeSelectionStore for ThemeFilePreferences<Files> {
    fn can_save_custom(&self, theme: &ThemeDefinition) -> bool {
        theme.kind() == &oblikovati_api::types::THEME_CUSTOM
            && self.sources.contains_key(theme.name())
    }
    fn save_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        self.save_custom(theme)
    }
    fn save_selection(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        self.selection.save_selection(theme)
    }
}
