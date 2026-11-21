// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, NativeAtomicFiles},
    load::PackageFileReader,
};
use std::{io, path::Path};

/// Native theme writes create the user config directory on the first save.
pub struct NativeThemePreferenceFiles;
impl PackageFileReader for NativeThemePreferenceFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        std::fs::read(path)
    }
}
impl AtomicFiles for NativeThemePreferenceFiles {
    type Stage = <NativeAtomicFiles as AtomicFiles>::Stage;
    fn stage(&mut self, path: &Path) -> io::Result<Self::Stage> {
        let parent = path
            .parent()
            .filter(|parent| !parent.as_os_str().is_empty())
            .unwrap_or(Path::new("."));
        std::fs::create_dir_all(parent)?;
        NativeAtomicFiles.stage(path)
    }
}

impl crate::theme_files::ThemeDirectoryFiles for NativeThemePreferenceFiles {
    fn theme_entries(&mut self, directory: &Path) -> io::Result<Vec<std::ffi::OsString>> {
        let entries = match std::fs::read_dir(directory) {
            Ok(entries) => entries,
            Err(cause) if cause.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
            Err(cause) => return Err(cause),
        };
        entries
            .map(|entry| entry.map(|entry| entry.file_name()))
            .collect()
    }
}
