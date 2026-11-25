// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    preference_revision::PreferenceRevision,
};
use oblikovati_api::wire::KeymapExport;
use oblikovati_model::keymap::{KeymapPreferenceAccess, KeymapPreferences};
use serde::{Deserialize, Serialize};
use std::{
    io,
    path::{Path, PathBuf},
};

/// Atomically stored keymap deltas in the Go application's YAML format.
pub struct KeymapPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    revision: PreferenceRevision,
    settings: StoredKeymap,
}

#[derive(Default, Deserialize, Serialize)]
struct StoredKeymap {
    #[serde(flatten)]
    delta: KeymapExport,
    #[serde(default, rename = "default-multi-character-aliases")]
    default_aliases: bool,
}
impl<Files: PackageFileReader + AtomicFiles> KeymapPreferenceFile<Files> {
    /// Read an existing keymap or start empty; example: `KeymapPreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let revision =
            PreferenceRevision::read(path, &mut files).map_err(|error| failure(path, error))?;
        let settings = serde_yaml_ng::from_slice::<Option<StoredKeymap>>(revision.contents())
            .map_err(|error| failure(path, error))?
            .unwrap_or_default();
        Ok(Self {
            path: path.into(),
            files,
            revision,
            settings,
        })
    }
}
impl<Files: PackageFileReader + AtomicFiles> KeymapPreferenceAccess
    for KeymapPreferenceFile<Files>
{
    fn keymap(&self) -> KeymapExport {
        self.settings.delta.clone()
    }
    fn set_keymap(&mut self, delta: &KeymapExport) -> Result<(), String> {
        self.set_preferences(&KeymapPreferences {
            delta: delta.clone(),
            default_aliases: self.settings.default_aliases,
        })
    }
    fn default_aliases(&self) -> bool {
        self.settings.default_aliases
    }
    fn set_preferences(&mut self, preferences: &KeymapPreferences) -> Result<(), String> {
        let candidate = StoredKeymap {
            delta: preferences.delta.clone(),
            default_aliases: preferences.default_aliases,
        };
        let encoded = crate::preference_yaml_guard::encode(&candidate)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision
            .require_current(&self.path, &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        replace_file(&self.path, encoded.as_bytes(), &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision = PreferenceRevision::Contents(encoded.into_bytes());
        self.settings = candidate;
        Ok(())
    }
}
fn failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "keymap preferences {path:?}: expected readable and writable YAML: {cause}"
    ))
}
