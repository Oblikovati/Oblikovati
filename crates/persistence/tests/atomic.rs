// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::atomic::{AtomicFiles, StagedFile, replace_file};
use std::{cell::RefCell, io, path::Path, rc::Rc};

#[derive(Default)]
struct FakeDisk {
    committed: Vec<u8>,
    calls: Vec<&'static str>,
    fail_at: Option<&'static str>,
}

struct FakeAtomicFiles(Rc<RefCell<FakeDisk>>);
struct FakeStage {
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

#[test]
fn replacement_orders_write_and_sync_before_commit() {
    let disk = Rc::new(RefCell::new(FakeDisk::default()));
    replace_file(
        Path::new("part.obk"),
        b"new",
        &mut FakeAtomicFiles(disk.clone()),
    )
    .unwrap();
    assert_eq!(disk.borrow().committed, b"new");
    assert_eq!(
        disk.borrow().calls,
        ["stage", "write", "sync", "commit", "drop"]
    );
}

#[test]
fn each_failure_preserves_the_old_file_and_stops_the_pipeline() {
    for phase in ["stage", "write", "sync", "commit"] {
        let disk = Rc::new(RefCell::new(FakeDisk {
            committed: b"old".to_vec(),
            fail_at: Some(phase),
            ..Default::default()
        }));
        let error = replace_file(
            Path::new("part.obk"),
            b"new",
            &mut FakeAtomicFiles(disk.clone()),
        )
        .unwrap_err();
        assert_eq!(error.phase, phase);
        assert_eq!(error.destination, Path::new("part.obk"));
        assert!(error.to_string().contains("part.obk"));
        assert!(std::error::Error::source(&error).is_some());
        assert_eq!(disk.borrow().committed, b"old");
        let calls = disk.borrow().calls.clone();
        let operations: Vec<_> = calls.iter().filter(|call| **call != "drop").collect();
        assert_eq!(operations.last().copied(), Some(&phase));
        assert_eq!(calls.contains(&"drop"), phase != "stage");
    }
}
