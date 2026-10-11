// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    load::PackageFileReader,
};
use std::{cell::RefCell, io, path::Path, rc::Rc};
#[derive(Default)]
pub struct FakeDisk {
    pub committed: Vec<u8>,
    pub calls: Vec<&'static str>,
    pub fail_at: Option<&'static str>,
    pub read_error: Option<io::ErrorKind>,
}

pub struct FakeAtomicFiles(pub Rc<RefCell<FakeDisk>>);
pub struct FakeStage {
    disk: Rc<RefCell<FakeDisk>>,
    staged: Vec<u8>,
}

fn attempt(disk: &Rc<RefCell<FakeDisk>>, phase: &'static str) -> io::Result<()> {
    let mut disk = disk.borrow_mut();
    disk.calls.push(phase);
    if disk.fail_at == Some(phase) {
        return Err(io::Error::other(format!("injected {phase} failure")));
    }
    Ok(())
}

impl AtomicFiles for FakeAtomicFiles {
    type Stage = FakeStage;
    fn stage(&mut self, _destination: &Path) -> io::Result<Self::Stage> {
        attempt(&self.0, "stage")?;
        Ok(FakeStage {
            disk: self.0.clone(),
            staged: Vec::new(),
        })
    }
}

impl StagedFile for FakeStage {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        self.staged.extend_from_slice(contents);
        attempt(&self.disk, "write")
    }
    fn synchronize(&mut self) -> io::Result<()> {
        attempt(&self.disk, "sync")
    }
    fn commit(self, _destination: &Path) -> io::Result<()> {
        attempt(&self.disk, "commit")?;
        self.disk.borrow_mut().committed = self.staged.clone();
        Ok(())
    }
}

impl Drop for FakeStage {
    fn drop(&mut self) {
        self.disk.borrow_mut().calls.push("drop");
    }
}

impl PackageFileReader for FakeAtomicFiles {
    fn read(&mut self, _path: &Path) -> io::Result<Vec<u8>> {
        attempt(&self.0, "read")?;
        let disk = self.0.borrow();
        if let Some(kind) = disk.read_error {
            return Err(io::Error::new(kind, "injected read decline"));
        }
        Ok(disk.committed.clone())
    }
}
