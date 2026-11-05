// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    atomic::{AtomicFiles, NativeAtomicFiles},
    load::{NativePackageReader, PackageFileReader},
    project::{NativeProjectFileProbe, ProjectFileProbe},
};
use std::{
    io,
    path::{Path, PathBuf},
};

/// Filesystem boundary for canonical-path lookup, reads, and atomic replacement.
pub trait DocumentFiles: AtomicFiles + PackageFileReader {
    /// Resolve a save destination, permitting a new file under an existing parent.
    /// Example: `let path = files.save_path(requested)?;` before checking workspace ownership.
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf>;
    /// Resolve an existing file without application-specific case folding.
    /// Example: `let path = files.canonical_path(requested)?;`.
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf>;
}

/// Native access through the existing package reader and atomic writer.
pub struct NativeDocumentFiles;
impl ProjectFileProbe for NativeDocumentFiles {
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool> {
        NativeProjectFileProbe.is_regular_file(path)
    }
}
impl PackageFileReader for NativeDocumentFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        NativePackageReader.read(path)
    }
}
impl AtomicFiles for NativeDocumentFiles {
    type Stage = <NativeAtomicFiles as AtomicFiles>::Stage;
    fn stage(&mut self, path: &Path) -> io::Result<Self::Stage> {
        NativeAtomicFiles.stage(path)
    }
}
impl DocumentFiles for NativeDocumentFiles {
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        match std::fs::symlink_metadata(requested) {
            Ok(_) => existing_destination(requested),
            Err(error) if error.kind() == io::ErrorKind::NotFound => new_destination(requested),
            Err(error) => Err(error),
        }
    }
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        std::fs::canonicalize(requested)
    }
}

fn existing_destination(requested: &Path) -> io::Result<PathBuf> {
    if !std::fs::metadata(requested)?.is_file() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("destination {requested:?}: expected a regular file"),
        ));
    }
    std::fs::canonicalize(requested)
}

fn new_destination(requested: &Path) -> io::Result<PathBuf> {
    let name = requested.file_name().ok_or_else(|| {
        io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("destination {requested:?}: expected a file name"),
        )
    })?;
    let parent = requested
        .parent()
        .filter(|path| !path.as_os_str().is_empty())
        .unwrap_or(Path::new("."));
    Ok(std::fs::canonicalize(parent)?.join(name))
}
