// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    atomic::{AtomicFiles, NativeAtomicFiles},
    load::{NativePackageReader, PackageFileReader},
};
use std::{
    io,
    path::{Path, PathBuf},
};

/// Filesystem boundary for canonical-path lookup, reads, and atomic replacement.
pub trait DocumentFiles: AtomicFiles + PackageFileReader {
    /// Resolve an existing file without application-specific case folding.
    /// Example: `let path = files.canonical_path(requested)?;`.
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf>;
}

/// Native access through the existing package reader and atomic writer.
pub struct NativeDocumentFiles;
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
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        std::fs::canonicalize(requested)
    }
}
