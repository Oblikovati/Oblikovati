// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn failed_activation_is_quarantined_until_explicit_cleanup() {
    if isolated("shutdown::failed_activation_is_quarantined_until_explicit_cleanup") {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let mut harness = RegistryHarness::new();
    let path = rust_library(
        &compiler,
        "failed-activation",
        &["lifecycle", "activate_error"],
    );
    let id = load(&mut harness.registry, &path);
    verify_activation_quarantine(&mut harness.registry, &id);
    finish_harness(&mut harness);
    assert_lifecycle(&harness.registry, &id, AddInState::Inactive);
    assert_eq!(callback_reply(harness.callbacks).0, ffi::OBK_ERR);
}
fn verify_activation_quarantine(registry: &mut NativeAddInRegistry, id: &str) {
    assert!(unsafe { registry.activate(id) }.is_err());
    assert_lifecycle(registry, id, AddInState::NeedsCleanup);
    assert!(!registry.describe(id).unwrap().activated);
    assert!(registry.unregister(id).is_err());
    assert!(unsafe { registry.activate(id) }.is_err());
}

#[test]
fn failed_shutdown_stops_notifications_but_keeps_dispatch_for_cleanup_retry() {
    if isolated(
        "shutdown::failed_shutdown_stops_notifications_but_keeps_dispatch_for_cleanup_retry",
    ) {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let harness = RegistryHarness::new();
    let (mut harness, rust, go, path) = prepare_cleanup_retry(harness, &compiler);
    verify_failed_shutdown(&mut harness.registry, &rust, &go, &path);
    verify_cleanup_dispatch(harness.callbacks, &mut harness.owner, &harness.awake);
    finish_harness(&mut harness);
    assert_lifecycle(&harness.registry, &rust, AddInState::Inactive);
    assert_eq!(callback_reply(harness.callbacks).0, ffi::OBK_ERR);
}
fn verify_failed_shutdown(registry: &mut NativeAddInRegistry, rust: &str, go: &str, path: &Path) {
    let failures = unsafe { registry.shutdown() }.unwrap_err();
    assert_eq!(failures.len(), 1);
    assert!(failures[0].0.contains(rust));
    assert_eq!(
        registry.lifecycle_state(rust).unwrap(),
        AddInState::NeedsCleanup
    );
    assert_eq!(registry.lifecycle_state(go).unwrap(), AddInState::Inactive);
    assert!(unsafe { registry.notify(rust, &[]) }.is_err());
    assert!(unsafe { registry.activate(go) }.is_err());
    assert!(unsafe { registry.call_automation(go, "test", &[]) }.is_err());
    assert!(unsafe { registry.load(path) }.is_err());
    assert!(registry.unregister(rust).is_err());
}
fn verify_cleanup_dispatch(
    callbacks: (ffi::ObkHostCall, ffi::ObkHostFree),
    owner: &mut OwnerDispatch<RegistryWake>,
    awake: &Receiver<()>,
) {
    let worker = thread::spawn(move || callback_reply(callbacks));
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_eq!(owner.drain(1, &RegistryEcho).unwrap(), 1);
    assert_eq!(worker.join().unwrap(), (ffi::OBK_OK, b"ping".to_vec()));
}
pub(super) fn callback_reply((call, free): (ffi::ObkHostCall, ffi::ObkHostFree)) -> (i32, Vec<u8>) {
    let mut reply = std::ptr::null_mut();
    let mut length = 0;
    // SAFETY: exact API callbacks, readable input and aligned live output slots; copy then matching free.
    let status = unsafe {
        call(
            c"echo".as_ptr(),
            b"ping".as_ptr(),
            4,
            &mut reply,
            &mut length,
        )
    };
    assert!(!reply.is_null());
    assert!(length >= 0);
    let copied = unsafe { std::slice::from_raw_parts(reply, length as usize) }.to_vec();
    unsafe { free(reply) };
    (status, copied)
}

fn prepare_cleanup_retry(
    harness: RegistryHarness,
    compiler: &compiler::FixtureCompiler,
) -> (RegistryHarness, String, String, PathBuf) {
    let mut harness = harness;
    let path = rust_library(
        compiler,
        "cleanup-retry",
        &["lifecycle", "deactivate_once_error"],
    );
    let rust = load(&mut harness.registry, &path);
    let go = load(&mut harness.registry, &go_library(compiler));
    unsafe { harness.registry.activate(&rust) }.unwrap();
    harness.registry =
        activate_with_owner(harness.registry, &mut harness.owner, &harness.awake, &go);
    (harness, rust, go, path)
}

#[test]
fn shutdown_drains_real_foreign_cleanup_before_retiring_callbacks() {
    if isolated("shutdown::shutdown_drains_real_foreign_cleanup_before_retiring_callbacks") {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let mut harness = RegistryHarness::new();
    let path = rust_library(
        &compiler,
        "cleanup-callback",
        &["lifecycle", "deactivate_echo"],
    );
    let id = load(&mut harness.registry, &path);
    unsafe { harness.registry.activate(&id) }.unwrap();
    harness.registry = shutdown_with_owner(harness.registry, &mut harness.owner, &harness.awake);
    assert_lifecycle(&harness.registry, &id, AddInState::Inactive);
    assert_eq!(callback_reply(harness.callbacks).0, ffi::OBK_ERR);
    harness.owner.close();
}
fn shutdown_with_owner(
    mut registry: NativeAddInRegistry,
    owner: &mut OwnerDispatch<RegistryWake>,
    awake: &Receiver<()>,
) -> NativeAddInRegistry {
    let worker = thread::spawn(move || {
        unsafe { registry.shutdown() }.unwrap();
        registry
    });
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_eq!(owner.drain(1, &RegistryEcho).unwrap(), 1);
    worker.join().unwrap()
}
