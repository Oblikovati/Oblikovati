// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::identity::{DocumentGuid, GuidSource, IdentityTransitionError};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::DocumentFiles,
    load::PackageFileReader,
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
    rc::Rc,
    thread::{self, ThreadId},
};
#[derive(Clone, Default)]
pub struct MemoryDocumentFiles {
    pub saved: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
}
pub struct MemoryStage {
    saved: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
    bytes: Vec<u8>,
}
impl PackageFileReader for MemoryDocumentFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.saved.borrow().get(path).cloned().ok_or_else(|| {
            io::Error::new(
                io::ErrorKind::NotFound,
                format!("memory path {path:?}: expected committed package"),
            )
        })
    }
}
impl DocumentFiles for MemoryDocumentFiles {
    fn canonical_path(&mut self, path: &Path) -> io::Result<PathBuf> {
        Ok(path.into())
    }
    fn save_path(&mut self, path: &Path) -> io::Result<PathBuf> {
        Ok(path.into())
    }
}
impl AtomicFiles for MemoryDocumentFiles {
    type Stage = MemoryStage;
    fn stage(&mut self, _: &Path) -> io::Result<MemoryStage> {
        Ok(MemoryStage {
            saved: self.saved.clone(),
            bytes: Vec::new(),
        })
    }
}
impl StagedFile for MemoryStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> {
        self.bytes = bytes.into();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn commit(self, path: &Path) -> io::Result<()> {
        self.saved.borrow_mut().insert(path.into(), self.bytes);
        Ok(())
    }
}
pub struct OwnerThreadGuids {
    owner: ThreadId,
    next: u64,
}
impl Default for OwnerThreadGuids {
    fn default() -> Self {
        Self {
            owner: thread::current().id(),
            next: 1,
        }
    }
}
impl GuidSource for OwnerThreadGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        assert_eq!(
            thread::current().id(),
            self.owner,
            "GUIDs must stay on the document owner"
        );
        let guid = format!("00000000-0000-4000-8000-{:012x}", self.next)
            .parse()
            .unwrap();
        self.next += 1;
        Ok(guid)
    }
}
