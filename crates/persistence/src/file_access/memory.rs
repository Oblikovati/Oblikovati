// SPDX-License-Identifier: GPL-2.0-only
//! In-memory `DocumentFiles` store for browser WebAssembly sessions and isolated tests.

use super::DocumentFiles;
use crate::{
    atomic::{AtomicFiles, StagedFile},
    load::PackageFileReader,
    project::ProjectFileProbe,
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io,
    path::{Component, Path, PathBuf},
    rc::Rc,
};

/// In-memory document store implementing [`DocumentFiles`] with atomic staging semantics.
///
/// ```
/// use oblikovati_persistence::file_access::MemoryDocumentFiles;
/// let mut files = MemoryDocumentFiles::default();
/// files.insert("/web/Part1.opd", b"demo".to_vec()).unwrap();
/// assert!(files.contains("/web/Part1.opd"));
/// ```
#[derive(Clone, Default)]
pub struct MemoryDocumentFiles {
    entries: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
}

/// Staged in-memory write that replaces the target only when committed.
pub struct MemoryStagedFile {
    entries: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
    buffer: Vec<u8>,
    synchronized: bool,
}

impl MemoryDocumentFiles {
    /// Insert or replace raw bytes at a normalized path.
    /// Example: `files.insert("/web/Part1.opd", bytes)?`.
    pub fn insert(&mut self, path: impl AsRef<Path>, bytes: Vec<u8>) -> io::Result<PathBuf> {
        let normalized = normalize_virtual_path(path.as_ref())?;
        self.entries.borrow_mut().insert(normalized.clone(), bytes);
        Ok(normalized)
    }

    /// Borrow a copy of stored bytes for a normalized path, if present.
    /// Example: `let bytes = files.get("/web/Part1.opd");`.
    pub fn get(&self, path: impl AsRef<Path>) -> Option<Vec<u8>> {
        let normalized = normalize_virtual_path(path.as_ref()).ok()?;
        self.entries.borrow().get(&normalized).cloned()
    }

    /// Report whether a normalized path exists in the store.
    /// Example: `assert!(files.contains("/web/Part1.opd"));`.
    pub fn contains(&self, path: impl AsRef<Path>) -> bool {
        normalize_virtual_path(path.as_ref())
            .ok()
            .is_some_and(|key| self.entries.borrow().contains_key(&key))
    }

    /// Remove a stored entry and return its previous bytes, if present.
    /// Example: `files.remove("/web/Part1.opd");`.
    pub fn remove(&mut self, path: impl AsRef<Path>) -> Option<Vec<u8>> {
        let normalized = normalize_virtual_path(path.as_ref()).ok()?;
        self.entries.borrow_mut().remove(&normalized)
    }

    /// List all stored paths in sorted deterministic order.
    /// Example: `let paths = files.paths();`.
    pub fn paths(&self) -> Vec<PathBuf> {
        self.entries.borrow().keys().cloned().collect()
    }
}

impl ProjectFileProbe for MemoryDocumentFiles {
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool> {
        let normalized = normalize_virtual_path(path)?;
        Ok(self.entries.borrow().contains_key(&normalized))
    }
}

impl PackageFileReader for MemoryDocumentFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        let normalized = normalize_virtual_path(path)?;
        self.entries
            .borrow()
            .get(&normalized)
            .cloned()
            .ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::NotFound,
                    format!("memory file {normalized:?}: expected an existing entry"),
                )
            })
    }
}

impl AtomicFiles for MemoryDocumentFiles {
    type Stage = MemoryStagedFile;

    fn stage(&mut self, destination: &Path) -> io::Result<Self::Stage> {
        let _ = normalize_virtual_path(destination)?;
        Ok(MemoryStagedFile {
            entries: Rc::clone(&self.entries),
            buffer: Vec::new(),
            synchronized: false,
        })
    }
}

impl StagedFile for MemoryStagedFile {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        self.buffer.extend_from_slice(contents);
        Ok(())
    }

    fn synchronize(&mut self) -> io::Result<()> {
        self.synchronized = true;
        Ok(())
    }

    fn commit(self, destination: &Path) -> io::Result<()> {
        let normalized = normalize_virtual_path(destination)?;
        self.entries.borrow_mut().insert(normalized, self.buffer);
        Ok(())
    }

    fn commit_new(self, destination: &Path) -> io::Result<()> {
        let normalized = normalize_virtual_path(destination)?;
        let mut entries = self.entries.borrow_mut();
        if entries.contains_key(&normalized) {
            return Err(io::Error::new(
                io::ErrorKind::AlreadyExists,
                format!("memory file {normalized:?}: expected an unused destination"),
            ));
        }
        entries.insert(normalized, self.buffer);
        Ok(())
    }
}

impl DocumentFiles for MemoryDocumentFiles {
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        normalize_virtual_path(requested)
    }

    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        let normalized = normalize_virtual_path(requested)?;
        if !self.entries.borrow().contains_key(&normalized) {
            return Err(io::Error::new(
                io::ErrorKind::NotFound,
                format!("memory file {normalized:?}: expected an existing entry"),
            ));
        }
        Ok(normalized)
    }
}

fn virtual_root() -> &'static Path {
    #[cfg(windows)]
    {
        Path::new(r"C:\")
    }
    #[cfg(not(windows))]
    {
        Path::new("/")
    }
}

fn normalize_virtual_path(path: &Path) -> io::Result<PathBuf> {
    if path.file_name().is_none() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("memory path {path:?}: expected a path with a file name"),
        ));
    }
    let root = virtual_root();
    let mut normalized = root.to_path_buf();
    for component in path.components() {
        match component {
            Component::RootDir | Component::CurDir | Component::Prefix(_) => {}
            Component::ParentDir => {
                if normalized != root {
                    normalized.pop();
                }
            }
            Component::Normal(segment) => normalized.push(segment),
        }
    }
    if normalized.file_name().is_none() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("memory path {path:?}: expected a non-root file path"),
        ));
    }
    Ok(normalized)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::atomic::{create_file, replace_file};

    #[test]
    fn memory_files_round_trip_atomic_replace_and_noclobber() {
        let mut files = MemoryDocumentFiles::default();
        let path = Path::new("/web/sub/../Part1.opd");
        let saved = files.save_path(path).unwrap();
        assert_eq!(saved, virtual_root().join("web").join("Part1.opd"));
        #[cfg(not(target_arch = "wasm32"))]
        assert!(saved.is_absolute(), "expected absolute path, got {saved:?}");
        assert!(files.canonical_path(&saved).is_err());

        create_file(path, b"first", &mut files).unwrap();
        assert!(files.is_regular_file(&saved).unwrap());
        assert_eq!(files.canonical_path(path).unwrap(), saved);
        assert_eq!(files.read(&saved).unwrap(), b"first");
        assert_eq!(files.get(&saved).unwrap(), b"first");
        assert_eq!(files.paths(), vec![saved.clone()]);

        let err = create_file(&saved, b"second", &mut files).unwrap_err();
        assert_eq!(err.kind(), io::ErrorKind::AlreadyExists);

        replace_file(&saved, b"updated", &mut files).unwrap();
        assert_eq!(files.read(&saved).unwrap(), b"updated");
        assert_eq!(files.remove(&saved).unwrap(), b"updated");
        assert!(!files.contains(&saved));
        assert!(files.read(&saved).is_err());
    }

    #[test]
    fn memory_files_reject_empty_and_root_paths() {
        let mut files = MemoryDocumentFiles::default();
        assert!(files.insert("", vec![]).is_err());
        assert!(files.insert("/", vec![]).is_err());
        assert!(files.stage(Path::new("/")).is_err());
    }
}
