// SPDX-License-Identifier: GPL-2.0-only
//! The Go-compatible global orientation preference file.
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
};
use oblikovati_model::orientation_preferences::OrientationPreferenceAccess;
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
};

#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase")]
struct Recipe {
    lock_to_selection: bool,
    #[serde(flatten)]
    retained: BTreeMap<String, serde_yaml_ng::Value>,
}

/// File-backed preference with injected I/O and atomic updates.
pub struct OrientationPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    locked: bool,
}
impl<Files: PackageFileReader + AtomicFiles> OrientationPreferenceFile<Files> {
    /// Load the saved preference; example: `OrientationPreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let recipe = read_recipe(path, &mut files)?;
        Ok(Self {
            path: path.into(),
            files,
            locked: recipe.lock_to_selection,
        })
    }
}
impl<Files: PackageFileReader + AtomicFiles> OrientationPreferenceAccess
    for OrientationPreferenceFile<Files>
{
    fn orientation_lock(&self) -> bool {
        self.locked
    }
    fn set_orientation_lock(&mut self, locked: bool) -> Result<(), String> {
        let mut recipe =
            read_recipe(&self.path, &mut self.files).map_err(|error| error.to_string())?;
        recipe.lock_to_selection = locked;
        let bytes =
            crate::preference_yaml_guard::encode(&recipe).map_err(|error| error.to_string())?;
        replace_file(&self.path, bytes.as_bytes(), &mut self.files)
            .map_err(|error| error.to_string())?;
        self.locked = locked;
        Ok(())
    }
}
fn read_recipe(path: &Path, files: &mut impl PackageFileReader) -> io::Result<Recipe> {
    let encoded = match files.read(path) {
        Ok(encoded) => encoded,
        Err(error) if error.kind() == io::ErrorKind::NotFound => Vec::new(),
        Err(error) => {
            return Err(io::Error::other(format!(
                "orientation preferences {path:?}: expected readable file: {error}"
            )));
        }
    };
    serde_yaml_ng::from_slice::<Option<Recipe>>(&encoded)
        .map(Option::unwrap_or_default)
        .map_err(|error| {
            io::Error::other(format!(
                "orientation preferences {path:?}: expected valid YAML: {error}"
            ))
        })
}
