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
    directory: Option<PathBuf>,
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
            directory: None,
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
impl<Files: AtomicFiles + PackageFileReader + crate::theme_lifecycle::ThemeLifecycleFiles>
    ThemeSelectionStore for ThemeFilePreferences<Files>
{
    fn can_create_custom(&self) -> bool {
        self.directory.is_some()
    }
    fn create_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        self.create_custom(theme)
    }
    fn remove_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        self.remove_custom(theme)
    }
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

impl<Files: AtomicFiles + PackageFileReader + crate::theme_lifecycle::ThemeLifecycleFiles>
    ThemeFilePreferences<Files>
{
    /// Permit creation under the configured directory; example: `store.with_custom_directory(root.join("themes"))`.
    pub fn with_custom_directory(mut self, directory: PathBuf) -> Self {
        self.directory = Some(directory);
        self
    }
    /// Publish a unique file before recording its identity; example: `store.create_custom(&copy)?`.
    pub fn create_custom(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        if theme.kind() != &oblikovati_api::types::THEME_CUSTOM
            || self.sources.contains_key(theme.name())
        {
            return Err(format!(
                "theme {:?}: expected a new custom name without a stored source",
                theme.name()
            ));
        }
        let directory = self.directory.as_ref().ok_or_else(|| {
            format!(
                "theme {:?}: expected a configured custom directory",
                theme.name()
            )
        })?;
        let path = self
            .custom_files
            .create_theme(directory, &theme.to_xml()?)
            .map_err(|cause| {
                format!(
                    "theme {:?}: expected a complete new XML file in {directory:?}: {cause}",
                    theme.name()
                )
            })?;
        self.sources.insert(theme.name().into(), path);
        Ok(())
    }
    /// Delete the original loaded file, keeping identity on failure; example: `store.remove_custom(&theme)?`.
    pub fn remove_custom(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        if theme.kind() != &oblikovati_api::types::THEME_CUSTOM {
            return Err(format!(
                "theme {:?}: expected a removable custom theme",
                theme.name()
            ));
        }
        let source = self.sources.get(theme.name()).ok_or_else(|| {
            format!(
                "theme {:?}: expected a known loaded XML source",
                theme.name()
            )
        })?;
        self.custom_files.remove_theme(source).map_err(|cause| {
            format!(
                "theme {:?}: expected removable XML source {source:?}: {cause}",
                theme.name()
            )
        })?;
        self.sources.remove(theme.name());
        Ok(())
    }
}
