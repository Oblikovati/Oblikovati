// SPDX-License-Identifier: GPL-2.0-only

use super::{AtomicFiles, StagedFile};
use std::{
    io::{self, Write},
    path::Path,
};
use tempfile::NamedTempFile;

/// Native sibling-temp replacement, using the platform's rename implementation.
pub struct NativeAtomicFiles;

/// A native staging file that removes itself if writing or replacement fails.
pub struct NativeStagedFile(NamedTempFile);

impl AtomicFiles for NativeAtomicFiles {
    type Stage = NativeStagedFile;

    fn stage(&mut self, destination: &Path) -> io::Result<Self::Stage> {
        if destination.file_name().is_none() {
            return Err(io::Error::new(
                io::ErrorKind::InvalidInput,
                format!("destination {destination:?}: expected a file name"),
            ));
        }
        let parent = destination
            .parent()
            .filter(|path| !path.as_os_str().is_empty())
            .unwrap_or_else(|| Path::new("."));
        tempfile::Builder::new()
            .prefix(".oblikovati-save-")
            .tempfile_in(parent)
            .map(NativeStagedFile)
    }
}

impl StagedFile for NativeStagedFile {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        self.0.write_all(contents)
    }

    fn synchronize(&mut self) -> io::Result<()> {
        apply_document_permissions(self.0.as_file())?;
        self.0.as_file().sync_all()
    }

    fn commit(self, destination: &Path) -> io::Result<()> {
        self.0
            .persist(destination)
            .map(|_file| ())
            .map_err(|failure| failure.error)
    }
}

fn apply_document_permissions(file: &std::fs::File) -> io::Result<()> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        // Match Go persistence's document mode; apply it before synchronization.
        file.set_permissions(std::fs::Permissions::from_mode(0o644))?;
    }
    #[cfg(not(unix))]
    let _ = file;
    Ok(())
}
