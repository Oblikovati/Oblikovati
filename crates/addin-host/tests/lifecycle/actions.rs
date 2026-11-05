// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn successful_lifecycle_serializes_wrapper_operations_and_copies_reply_before_free() {
    let compiler = FixtureCompiler::new();
    let mut addin = fixture_addin(&compiler, "success", &["automation"]);
    assert_eq!(addin.state(), AddInState::Inactive);
    assert_eq!(addin.metadata().id, "rust-fixture");
    assert!(automation(&mut addin, "echo", b"blocked").is_err());
    activate(&mut addin).unwrap();
    assert!(activate(&mut addin).is_err());
    assert_eq!(
        automation(&mut addin, "echo", b"owned-copy").unwrap(),
        b"owned-copy"
    );
    assert_eq!(free_count(&mut addin), 1);
    notify_sizes(&mut addin);
    deactivation(&mut addin).unwrap();
    deactivation(&mut addin).unwrap();
    assert_eq!(addin.state(), AddInState::Inactive);
    assert!(automation(&mut addin, "echo", b"").is_err());
}
fn notify_sizes(addin: &mut LoadedAddIn) {
    // SAFETY: fixture synchronously validates borrowed slice/null/length shapes, without retaining them.
    unsafe {
        addin.notify(&[]).unwrap();
        addin
            .notify(br#"{"type":"selection.changed","count":0}"#)
            .unwrap();
    }
}
#[test]
fn malformed_and_error_replies_are_freed_once_without_invalid_rust_references() {
    let compiler = FixtureCompiler::new();
    let mut addin = fixture_addin(&compiler, "replies", &["automation"]);
    activate(&mut addin).unwrap();
    for (method, expected) in [
        ("negative", "expected nonnegative length"),
        ("error", "fixture refused"),
        ("bad_utf8", "expected UTF-8"),
    ] {
        verify_retired_error(&mut addin, method, expected);
    }
    let before = free_count(&mut addin);
    assert!(automation(&mut addin, "null_nonzero", &[]).is_err());
    assert_eq!(free_count(&mut addin), before + 1);
    deactivation(&mut addin).unwrap();
}
#[test]
fn empty_replies_and_zero_length_allocations_have_distinct_release_ownership() {
    let compiler = FixtureCompiler::new();
    let path = compiler.rust_library("empty", &["lifecycle", "automation"], API_MAJOR, API_MINOR);
    let mut addin = open(&path);
    activate(&mut addin).unwrap();
    let before = free_count(&mut addin);
    assert_eq!(
        automation(&mut addin, "empty", &[]).unwrap(),
        Vec::<u8>::new()
    );
    assert_eq!(free_count(&mut addin), before + 1);
    assert_eq!(
        automation(&mut addin, "zero", &[]).unwrap(),
        Vec::<u8>::new()
    );
    assert_eq!(free_count(&mut addin), before + 3);
    deactivation(&mut addin).unwrap();
}
#[test]
fn failed_activation_requires_cleanup_and_failed_deactivation_refuses_further_work() {
    let compiler = FixtureCompiler::new();
    for (name, flag) in [
        ("activation", "activate_error"),
        ("deactivation", "deactivate_error"),
    ] {
        verify_lifecycle_failure(&compiler, name, flag);
    }
}
fn verify_lifecycle_failure(compiler: &FixtureCompiler, name: &str, flag: &str) {
    let mut addin = fixture_addin(compiler, name, &["automation", flag]);
    if flag == "activate_error" {
        assert!(activate(&mut addin).is_err());
    } else {
        activate(&mut addin).unwrap();
        assert!(deactivation(&mut addin).is_err());
    }
    assert_eq!(addin.state(), AddInState::NeedsCleanup);
    assert!(activate(&mut addin).is_err());
    assert!(automation(&mut addin, "echo", b"").is_err());
    // SAFETY: attempted fixture notification must be refused before calling native code.
    assert!(unsafe { addin.notify(&[]) }.is_err());
    if flag == "activate_error" {
        deactivation(&mut addin).unwrap();
        assert_eq!(addin.state(), AddInState::Inactive);
    }
}
#[test]
fn absent_automation_nul_methods_and_failed_notify_are_named_declines() {
    let compiler = FixtureCompiler::new();
    let mut addin = fixture_addin(&compiler, "optional", &["notify_error"]);
    activate(&mut addin).unwrap();
    assert!(
        automation(&mut addin, "echo", &[])
            .unwrap_err()
            .to_string()
            .contains("optional ObkAddInAutomation")
    );
    // SAFETY: fixture reports notification failure without retaining borrowed data.
    assert!(
        unsafe { addin.notify(b"event") }
            .unwrap_err()
            .to_string()
            .contains("returned 1")
    );
    deactivation(&mut addin).unwrap();
    verify_nul_method(&compiler);
}
fn verify_nul_method(compiler: &FixtureCompiler) {
    let path = compiler.rust_library("nul", &["lifecycle", "automation"], API_MAJOR, API_MINOR);
    let mut addin = open(&path);
    activate(&mut addin).unwrap();
    assert!(
        automation(&mut addin, "bad\0method", &[])
            .unwrap_err()
            .to_string()
            .contains("expected no embedded NUL")
    );
    deactivation(&mut addin).unwrap();
}

fn fixture_addin(compiler: &FixtureCompiler, name: &str, extra: &[&str]) -> LoadedAddIn {
    let mut flags = vec!["lifecycle"];
    flags.extend_from_slice(extra);
    let path = compiler.rust_library(name, &flags, API_MAJOR, API_MINOR);
    open(&path)
}
fn verify_retired_error(addin: &mut LoadedAddIn, method: &str, expected: &str) {
    let before = free_count(addin);
    let error = automation(addin, method, b"reply").unwrap_err().to_string();
    assert!(error.contains(expected));
    // The previous counter query owns a reply too: one query release plus one rejected automation release.
    assert_eq!(free_count(addin), before + 2);
}
