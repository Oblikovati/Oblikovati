// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
};
use oblikovati_model::application_options::{SketchApplicationOptions, SketchPreferenceAccess};
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
    // Other Go groups stay opaque until their owners are ported; a sketch edit must preserve them.
    #[serde(flatten)]
    retained_groups: BTreeMap<String, serde_yaml_ng::Value>,
}
/// One application options file, independent of document save and undo state.
pub struct SketchPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    recipe: PreferenceRecipe,
}
impl<Files: PackageFileReader + AtomicFiles> SketchPreferenceFile<Files> {
    /// Load stored overrides over defaults; an absent or empty file uses defaults.
    /// Example: `SketchPreferenceFile::load(path, native_files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let encoded = match files.read(path) {
            Ok(encoded) => encoded,
            Err(error) if error.kind() == io::ErrorKind::NotFound => Vec::new(),
            Err(error) => return Err(preference_failure(path, error)),
        };
        let recipe = decode_preferences(path, &encoded)?;
        Ok(Self {
            path: path.into(),
            files,
            recipe,
        })
    }
    /// Save and publish validated options after atomic replacement.
    /// Example: `preferences.save_options(updated)?`.
    pub fn save_options(&mut self, options: SketchApplicationOptions) -> io::Result<()> {
        options
            .validate()
            .map_err(|cause| preference_failure(&self.path, cause))?;
        let mut candidate = self.recipe.clone();
        candidate.sketch = options;
        let encoded = serde_yaml_ng::to_string(&candidate)
            .map_err(|cause| preference_failure(&self.path, cause))?;
        replace_file(&self.path, encoded.as_bytes(), &mut self.files).map_err(io::Error::other)?;
        self.recipe = candidate;
        Ok(())
    }
}
impl<Files: PackageFileReader + AtomicFiles> SketchPreferenceAccess
    for SketchPreferenceFile<Files>
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
        .sketch
        .validate()
        .map_err(|cause| preference_failure(path, cause))?;
    Ok(recipe)
}
fn preference_failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "application options {path:?}: expected readable, valid sketch preferences: {cause}"
    ))
}
