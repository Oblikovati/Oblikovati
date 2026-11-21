// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    preference_revision::PreferenceRevision,
};
use oblikovati_api::wire::KeymapExport;
use oblikovati_model::keymap::KeymapPreferenceAccess;
use std::{
    io,
    path::{Path, PathBuf},
};

/// Atomically stored keymap deltas in the Go application's YAML format.
pub struct KeymapPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    revision: PreferenceRevision,
    delta: KeymapExport,
}
impl<Files: PackageFileReader + AtomicFiles> KeymapPreferenceFile<Files> {
    /// Read an existing keymap or start empty; example: `KeymapPreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let revision =
            PreferenceRevision::read(path, &mut files).map_err(|error| failure(path, error))?;
        let delta = serde_yaml_ng::from_slice::<Option<KeymapExport>>(revision.contents())
            .map_err(|error| failure(path, error))?
            .unwrap_or_default();
        Ok(Self {
            path: path.into(),
            files,
            revision,
            delta,
        })
    }
}
impl<Files: PackageFileReader + AtomicFiles> KeymapPreferenceAccess
    for KeymapPreferenceFile<Files>
{
    fn keymap(&self) -> KeymapExport {
        self.delta.clone()
    }
    fn set_keymap(&mut self, delta: &KeymapExport) -> Result<(), String> {
        let encoded = crate::preference_yaml_guard::encode(delta)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision
            .require_current(&self.path, &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        replace_file(&self.path, encoded.as_bytes(), &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision = PreferenceRevision::Contents(encoded.into_bytes());
        self.delta = delta.clone();
        Ok(())
    }
}
fn failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "keymap preferences {path:?}: expected readable and writable YAML: {cause}"
    ))
}
