// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
};
use oblikovati_model::application_options::{SketchApplicationOptions, SketchPreferenceAccess};
use oblikovati_model::ui_scale::{UiScaleOptions, UiScalePreferenceAccess};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
};

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
struct PreferenceRecipe {
    sketch: SketchApplicationOptions,
    #[serde(skip_serializing_if = "Option::is_none")]
    ui: Option<UiPreferenceRecipe>,
    // Other Go groups stay opaque until their owners are ported; group edits preserve them.
    #[serde(flatten)]
    retained_groups: BTreeMap<String, serde_yaml_ng::Value>,
}
/// One application options file, independent of document save and undo state.
pub struct ApplicationPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    recipe: PreferenceRecipe,
    ui_scale: UiScaleOptions,
}
impl<Files: PackageFileReader + AtomicFiles> ApplicationPreferenceFile<Files> {
    /// Load stored overrides over defaults; an absent or empty file uses defaults.
    /// Example: `SketchPreferenceFile::load(path, native_files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let recipe = read_preferences(path, &mut files)?;
        let ui_scale = recipe
            .ui_options()
            .map_err(|cause| preference_failure(path, cause))?;
        Ok(Self {
            path: path.into(),
            files,
            recipe,
            ui_scale,
        })
    }
    /// Save and publish validated options after atomic replacement.
    /// Example: `preferences.save_options(updated)?`.
    pub fn save_options(&mut self, options: SketchApplicationOptions) -> io::Result<()> {
        options
            .validate()
            .map_err(|cause| preference_failure(&self.path, cause))?;
        let mut candidate = read_preferences(&self.path, &mut self.files)?;
        candidate.sketch = options;
        self.publish(candidate)
    }
    /// Save text/icon scale without changing other groups; example: `preferences.save_ui_scale(options)?`.
    pub fn save_ui_scale(&mut self, options: UiScaleOptions) -> io::Result<()> {
        let mut candidate = read_preferences(&self.path, &mut self.files)?;
        let ui = candidate.ui.get_or_insert_default();
        ui.font_scale = Some(options.font());
        ui.icon_scale = Some(options.icon());
        self.publish(candidate)
    }
    fn publish(&mut self, candidate: PreferenceRecipe) -> io::Result<()> {
        let ui_scale = candidate
            .ui_options()
            .map_err(|cause| preference_failure(&self.path, cause))?;
        let encoded = serde_yaml_ng::to_string(&candidate)
            .map_err(|cause| preference_failure(&self.path, cause))?;
        replace_file(&self.path, encoded.as_bytes(), &mut self.files).map_err(io::Error::other)?;
        self.recipe = candidate;
        self.ui_scale = ui_scale;
        Ok(())
    }
}
impl<Files: PackageFileReader + AtomicFiles> SketchPreferenceAccess
    for ApplicationPreferenceFile<Files>
{
    fn sketch_options(&self) -> &SketchApplicationOptions {
        &self.recipe.sketch
    }
    fn set_sketch_options(&mut self, options: SketchApplicationOptions) -> Result<(), String> {
        self.save_options(options)
            .map_err(|cause| cause.to_string())
    }
}
fn decode_preferences(path: &Path, encoded: &[u8]) -> io::Result<PreferenceRecipe> {
    let recipe: PreferenceRecipe = serde_yaml_ng::from_slice::<Option<PreferenceRecipe>>(encoded)
        .map_err(|cause| preference_failure(path, cause))?
        .unwrap_or_default();
    recipe
        .ui_options()
        .map_err(|cause| preference_failure(path, cause))?;
    recipe
        .sketch
        .validate()
        .map_err(|cause| preference_failure(path, cause))?;
    Ok(recipe)
}
fn preference_failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "application options {path:?}: expected readable, valid sketch and UI preferences: {cause}"
    ))
}

/// Compatibility name for existing sketch-only callers; both groups now share one owner.
pub type SketchPreferenceFile<Files> = ApplicationPreferenceFile<Files>;

impl<Files: PackageFileReader + AtomicFiles> UiScalePreferenceAccess
    for ApplicationPreferenceFile<Files>
{
    fn ui_scale(&self) -> UiScaleOptions {
        self.ui_scale
    }
    fn set_ui_scale(&mut self, options: UiScaleOptions) -> Result<(), String> {
        self.save_ui_scale(options)
            .map_err(|cause| cause.to_string())
    }
}
fn read_preferences(
    path: &Path,
    files: &mut impl PackageFileReader,
) -> io::Result<PreferenceRecipe> {
    let encoded = match files.read(path) {
        Ok(encoded) => encoded,
        Err(error) if error.kind() == io::ErrorKind::NotFound => Vec::new(),
        Err(error) => return Err(preference_failure(path, error)),
    };
    decode_preferences(path, &encoded)
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct UiPreferenceRecipe {
    #[serde(skip_serializing_if = "Option::is_none")]
    font_scale: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    icon_scale: Option<f32>,
    // Go also owns menu/status visibility, and future keys must round-trip unchanged.
    #[serde(flatten)]
    retained: BTreeMap<String, serde_yaml_ng::Value>,
}
impl UiPreferenceRecipe {
    fn options(&self) -> Result<UiScaleOptions, String> {
        UiScaleOptions::from_stored(self.font_scale.unwrap_or(1.), self.icon_scale.unwrap_or(1.))
    }
}

impl PreferenceRecipe {
    fn ui_options(&self) -> Result<UiScaleOptions, String> {
        self.ui
            .as_ref()
            .map(UiPreferenceRecipe::options)
            .transpose()
            .map(Option::unwrap_or_default)
    }
}
