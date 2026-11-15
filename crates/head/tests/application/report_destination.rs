// SPDX-License-Identifier: GPL-2.0-only
//! Reserve empty report destinations without overwriting earlier evidence.
use std::{
    cell::RefCell,
    fs,
    path::{Path, PathBuf},
};

thread_local! {
    static DEFAULT_REPORTS: RefCell<Option<PathBuf>> = const { RefCell::new(None) };
}

pub(super) fn default_folder() -> PathBuf {
    DEFAULT_REPORTS.with(|slot| {
        let mut slot = slot.borrow_mut();
        slot.get_or_insert_with(create_default_folder).clone()
    })
}

fn create_default_folder() -> PathBuf {
    let parent = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../target/benchmarks/application");
    fs::create_dir_all(&parent).expect("BDD reports: expected report parent directory");
    tempfile::Builder::new()
        .prefix("run-")
        .tempdir_in(parent)
        .expect("BDD reports: expected unique report directory")
        .keep()
}

pub(super) fn create_file(path: &Path) -> std::io::Result<fs::File> {
    fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
}
pub(super) fn write_file(path: &Path, contents: &[u8]) -> std::io::Result<()> {
    use std::io::Write;
    create_file(path)?.write_all(contents)
}

pub(super) struct ReportLease(PathBuf);
impl Drop for ReportLease {
    fn drop(&mut self) {
        let _ = fs::remove_file(&self.0);
    }
}

pub(super) fn reserve(folder: &Path) -> Result<ReportLease, String> {
    fs::create_dir_all(folder).map_err(|error| format!("BDD reports {folder:?}: {error}"))?;
    let lock = folder.join(".oblikovati-bdd-run");
    fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&lock)
        .map_err(|error| {
            format!("BDD reports {folder:?}: expected exclusive empty directory: {error}")
        })?;
    let lease = ReportLease(lock);
    ensure_empty(folder, &lease.0)?;
    Ok(lease)
}

fn ensure_empty(folder: &Path, lock: &Path) -> Result<(), String> {
    let entries =
        fs::read_dir(folder).map_err(|error| format!("BDD reports {folder:?}: {error}"))?;
    for entry in entries {
        let path = entry
            .map_err(|error| format!("BDD reports {folder:?}: {error}"))?
            .path();
        if path != lock {
            return Err(format!(
                "BDD reports {folder:?}: expected empty directory, found {path:?}"
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn default_reports_are_unique_and_survive_the_test_thread() {
        let first = std::thread::spawn(default_folder).join().unwrap();
        let second = std::thread::spawn(default_folder).join().unwrap();
        assert_ne!(first, second);
        assert!(first.is_dir() && second.is_dir());
        fs::remove_dir(first).unwrap();
        fs::remove_dir(second).unwrap();
    }
    #[test]
    fn a_concurrent_report_cannot_be_overwritten() {
        let folder = tempfile::tempdir().unwrap();
        let path = folder.path().join("report.json");
        write_file(&path, b"original").unwrap();
        assert!(write_file(&path, b"replacement").is_err());
        assert_eq!(fs::read(path).unwrap(), b"original");
    }
    #[test]
    fn occupied_destination_is_refused_without_removing_evidence() {
        let folder = tempfile::tempdir().unwrap();
        let evidence = folder.path().join("cucumber.json");
        fs::write(&evidence, b"prior evidence").unwrap();
        assert!(reserve(folder.path()).is_err());
        assert_eq!(fs::read(evidence).unwrap(), b"prior evidence");
        assert!(!folder.path().join(".oblikovati-bdd-run").exists());
    }
    #[test]
    fn reservation_excludes_another_run_without_removing_its_lock() {
        let folder = tempfile::tempdir().unwrap();
        let lease = reserve(folder.path()).unwrap();
        assert!(reserve(folder.path()).is_err());
        assert!(folder.path().join(".oblikovati-bdd-run").exists());
        drop(lease);
        assert!(reserve(folder.path()).is_ok());
    }
}
