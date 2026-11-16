// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
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
    delta: KeymapExport,
}
impl<Files: PackageFileReader + AtomicFiles> KeymapPreferenceFile<Files> {
    /// Read an existing keymap or start empty; example: `KeymapPreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let bytes = match files.read(path) {
            Ok(bytes) => bytes,
            Err(error) if error.kind() == io::ErrorKind::NotFound => Vec::new(),
            Err(error) => return Err(failure(path, error)),
        };
        let delta = serde_yaml_ng::from_slice::<Option<KeymapExport>>(&bytes)
            .map_err(|error| failure(path, error))?
            .unwrap_or_default();
        Ok(Self {
            path: path.into(),
            files,
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
        replace_file(&self.path, encoded.as_bytes(), &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.delta = delta.clone();
        Ok(())
    }
}
fn failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "keymap preferences {path:?}: expected readable and writable YAML: {cause}"
    ))
}
