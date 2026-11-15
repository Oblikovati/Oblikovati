// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the default benchmark to a verified, temporary pinned checkout.
use std::{
    cell::RefCell,
    ffi::OsString,
    path::{Path, PathBuf},
    process::Command,
};

thread_local! {
    static DEFAULT_CHECKOUT: RefCell<Option<tempfile::TempDir>> = const { RefCell::new(None) };
}

pub(super) fn root(explicit: Option<OsString>) -> PathBuf {
    if let Some(path) = explicit {
        return PathBuf::from(path);
    }
    DEFAULT_CHECKOUT.with(|slot| {
        let mut slot = slot.borrow_mut();
        let folder = slot.get_or_insert_with(restore_default);
        folder.path().join("checkout")
    })
}

fn restore_default() -> tempfile::TempDir {
    let directory = tempfile::Builder::new()
        .prefix("oblikovati-bdd-")
        .tempdir()
        .expect("scenario checkout: expected a temporary directory");
    verify_command(&directory.path().join("checkout"), true)
        .unwrap_or_else(|error| panic!("{error}"));
    directory
}

pub(super) fn verify_required(root: &Path, explicit: bool, strict: bool) -> Result<(), String> {
    if explicit && !strict {
        return Ok(());
    }
    verify_command(root, false)
}

pub(super) fn validate_gate_filters(strict: bool, configured: &[&str]) -> Result<(), String> {
    if strict && !configured.is_empty() {
        return Err(format!(
            "scenario filters {configured:?}: expected an unfiltered gate run"
        ));
    }
    Ok(())
}

pub(super) fn configured_filters() -> Vec<&'static str> {
    [
        "OBK_BDD_DIAGNOSTIC",
        "OBK_BDD_FEATURE",
        "OBK_BDD_SCENARIO",
        "OBK_BDD_AREA",
    ]
    .into_iter()
    .filter(|name| std::env::var_os(name).is_some())
    .collect()
}

fn verify_command(root: &Path, restore: bool) -> Result<(), String> {
    let result = verifier_command(restore)
        .arg(root)
        .output()
        .map_err(|error| {
            format!("scenario source {root:?}: expected Python snapshot verifier: {error}")
        })?;
    if !result.status.success() {
        return Err(format!(
            "scenario source {root:?}: expected pinned clean checkout: {}",
            String::from_utf8_lossy(&result.stderr)
        ));
    }
    Ok(())
}

fn verifier_command(restore: bool) -> Command {
    let repository = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let python = if cfg!(windows) { "python" } else { "python3" };
    let mut command = Command::new(python);
    command
        .arg("-B")
        .arg(repository.join("tools/bdd/verify_snapshot.py"));
    if restore {
        command.arg("--restore");
    }
    command
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strict_gate_refuses_filtered_or_diagnostic_runs() {
        assert!(validate_gate_filters(true, &[]).is_ok());
        assert!(validate_gate_filters(false, &["OBK_BDD_FEATURE"]).is_ok());
        assert!(validate_gate_filters(true, &["OBK_BDD_FEATURE"]).is_err());
        assert!(validate_gate_filters(true, &["OBK_BDD_DIAGNOSTIC"]).is_err());
    }

    #[test]
    fn explicit_diagnostic_sources_remain_available() {
        let directory = tempfile::tempdir().unwrap();
        assert_eq!(
            root(Some(directory.path().as_os_str().into())),
            directory.path()
        );
        assert!(verify_required(directory.path(), true, false).is_ok());
        let error = verify_required(directory.path(), true, true).unwrap_err();
        assert!(error.contains("expected pinned clean checkout"), "{error}");
    }

    #[test]
    fn pinned_verification_rejects_changed_and_missing_sources() {
        let folder = restore_default();
        let checkout = folder.path().join("checkout");
        verify_required(&checkout, true, true).unwrap();
        let feature =
            checkout.join("features/application/document-creation/document-creation.feature");
        std::fs::write(&feature, "Feature: modified\n").unwrap();
        assert!(verify_required(&checkout, true, true).is_err());
        assert!(verify_required(&folder.path().join("missing"), true, true).is_err());
    }
}
