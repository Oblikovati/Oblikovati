// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::atomic::replace_file;
use std::{cell::RefCell, path::Path, rc::Rc};
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
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
