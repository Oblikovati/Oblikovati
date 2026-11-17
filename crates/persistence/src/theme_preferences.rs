// SPDX-License-Identifier: GPL-2.0-only
pub use crate::theme_preferences_native::NativeThemePreferenceFiles;
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    preference_revision::PreferenceRevision,
};
use oblikovati_model::theme::{ThemeDefinition, ThemeSelectionStore};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
};

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(default)]
struct ThemePreferenceRecipe {
    #[serde(rename = "activeTheme")]
    active_theme: String,
    // The Go preferences file can contain groups whose Rust owners are not ported yet.
    #[serde(flatten)]
    retained_groups: BTreeMap<String, serde_yaml_ng::Value>,
}

/// Active theme storage is independent of Part save and undo history.
pub struct ThemePreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    revision: PreferenceRevision,
    recipe: ThemePreferenceRecipe,
}
impl<Files: PackageFileReader + AtomicFiles> ThemePreferenceFile<Files> {
    /// Load the Go activeTheme preference; example: `ThemePreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let revision = PreferenceRevision::read(path, &mut files)
            .map_err(|cause| theme_preference_failure(path, cause))?;
        let recipe =
            serde_yaml_ng::from_slice::<Option<ThemePreferenceRecipe>>(revision.contents())
                .map_err(|cause| theme_preference_failure(path, cause))?
                .unwrap_or_default();
        Ok(Self {
            path: path.into(),
            files,
            revision,
            recipe,
        })
    }
    /// Read the stored selection; example: `ThemeLibrary::new(customs, preferences.active_name())?`.
    pub fn active_name(&self) -> &str {
        &self.recipe.active_theme
    }
    /// Publish the selected name only after atomic save; example: `preferences.save_active(library.active())?`.
    pub fn save_active(&mut self, theme: &ThemeDefinition) -> io::Result<()> {
        let mut candidate = self.recipe.clone();
        candidate.active_theme = theme.name().into();
        let encoded = crate::preference_yaml_guard::encode(&candidate)
            .map_err(|cause| theme_preference_failure(&self.path, cause))?;
        self.revision.require_current(&self.path, &mut self.files)?;
        replace_file(&self.path, encoded.as_bytes(), &mut self.files).map_err(io::Error::other)?;
        self.revision = PreferenceRevision::Contents(encoded.into_bytes());
        self.recipe = candidate;
        Ok(())
    }
}
fn theme_preference_failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "theme preferences {path:?}: expected a readable YAML mapping with activeTheme: {cause}"
    ))
}

impl<Files: PackageFileReader + AtomicFiles> ThemeSelectionStore for ThemePreferenceFile<Files> {
    fn save_selection(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        self.save_active(theme).map_err(|cause| cause.to_string())
    }
}
