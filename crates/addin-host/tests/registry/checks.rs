// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn ordered_registry_ports_go_policy_automation_and_reload() {
    if isolated("checks::ordered_registry_ports_go_policy_automation_and_reload") {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let mut harness = RegistryHarness::new();
    let rust = load(
        &mut harness.registry,
        &rust_library(&compiler, "plain", &[]),
    );
    let go_path = go_library(&compiler);
    let go = load(&mut harness.registry, &go_path);
    verify_policies(&mut harness.registry, &rust, &go, &go_path);
    harness.registry =
        activate_with_owner(harness.registry, &mut harness.owner, &harness.awake, &go);
    verify_go_runtime(&mut harness.registry, &go);
    verify_reload(&mut harness.registry, &go, &go_path);
    finish_harness(&mut harness);
    verify_closed(&mut harness.registry, &go_path, &go, harness.callbacks);
}
fn verify_policies(registry: &mut NativeAddInRegistry, rust: &str, go: &str, path: &Path) {
    let contract: &dyn AddInRegistry = registry;
    assert_eq!(contract.registered(), Some(vec![rust.into(), go.into()]));
    assert_eq!(registry.startup_ids(), vec![rust.to_owned(), go.to_owned()]);
    registry.set_load_behavior(rust, LOAD_DISABLED).unwrap();
    registry.set_load_behavior(go, LOAD_ON_DEMAND).unwrap();
    assert!(registry.startup_ids().is_empty());
    assert!(
        unsafe { registry.activate(rust) }
            .unwrap_err()
            .0
            .contains("disabled")
    );
    assert!(unsafe { registry.call_automation(go, "echo", b"ping") }.is_err());
    verify_manifest(registry, go, path);
    assert_eq!(registry.load_behavior("unknown".into()), LOAD_ON_STARTUP);
}
fn verify_manifest(registry: &NativeAddInRegistry, go: &str, path: &Path) {
    let identity = registry.describe(go).unwrap();
    assert_eq!(identity.display_name, "Echo Fixture");
    assert_eq!(identity.location, path.to_str().unwrap());
    assert!(identity.has_automation);
    assert!(!identity.activated);
}
fn verify_go_runtime(registry: &mut NativeAddInRegistry, go: &str) {
    assert!(registry.is_active(go.into()));
    assert!(registry.describe(go).unwrap().activated);
    unsafe { registry.notify(go, b"{}") }.unwrap();
    let reply = unsafe { registry.call_automation(go, "test", b"ping") }.unwrap();
    assert_eq!(reply, br#"{"method":"test","payload":"ping"}"#);
    assert!(registry.unregister(go).is_err());
    unsafe { registry.deactivate(go) }.unwrap();
    unsafe { registry.deactivate(go) }.unwrap();
    assert!(!registry.is_active(go.into()));
}
fn verify_reload(registry: &mut NativeAddInRegistry, go: &str, path: &Path) {
    registry.unregister(go).unwrap();
    assert_eq!(load(registry, path), go);
    assert_eq!(registry.load_behavior(go.into()), LOAD_ON_DEMAND);
    assert_eq!(registry.registered().unwrap().last().unwrap(), go);
    assert!(registry.describe("missing").is_err());
    assert!(!registry.is_active("missing".into()));
    assert!(registry.lifecycle_state("missing").is_err());
    assert!(registry.unregister("missing").is_err());
    unsafe { registry.deactivate("missing") }.unwrap();
}
fn verify_closed(
    registry: &mut NativeAddInRegistry,
    path: &Path,
    id: &str,
    callbacks: (ffi::ObkHostCall, ffi::ObkHostFree),
) {
    assert!(unsafe { registry.load(path) }.is_err());
    assert!(unsafe { registry.activate(id) }.is_err());
    assert!(unsafe { registry.notify(id, b"{}") }.is_err());
    assert!(unsafe { registry.call_automation(id, "test", &[]) }.is_err());
    assert!(registry.set_load_behavior(id, LOAD_ON_STARTUP).is_err());
    assert_eq!(shutdown::callback_reply(callbacks).0, ffi::OBK_ERR);
    unsafe { registry.shutdown() }.unwrap();
}
#[test]
fn admission_refuses_duplicate_empty_and_incompatible_ids_without_replacement() {
    if isolated(
        "checks::admission_refuses_duplicate_empty_and_incompatible_ids_without_replacement",
    ) {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let mut harness = RegistryHarness::new();
    assert_eq!(harness.registry.registered(), Some(Vec::new()));
    let id = verify_malformed_manifest(&mut harness.registry, &compiler);
    verify_admission_errors(&mut harness.registry, &compiler, &id);
    finish_harness(&mut harness);
}
fn verify_malformed_manifest(
    registry: &mut NativeAddInRegistry,
    compiler: &compiler::FixtureCompiler,
) -> String {
    let malformed = rust_library(compiler, "bad-manifest", &["invalid_manifest"]);
    let id = load(registry, &malformed);
    assert!(registry.describe(&id).unwrap().display_name.is_empty());
    assert!(
        unsafe { registry.load(&malformed) }
            .unwrap_err()
            .0
            .contains("unique")
    );
    id
}

fn verify_admission_errors(
    registry: &mut NativeAddInRegistry,
    compiler: &compiler::FixtureCompiler,
    id: &str,
) {
    assert!(
        registry
            .set_load_behavior(id, AddInLoadBehavior(255))
            .is_err()
    );
    assert!(
        registry
            .set_load_behavior("missing", LOAD_ON_STARTUP)
            .is_err()
    );
    verify_rejected_libraries(registry, compiler);
    assert_eq!(registry.registered(), Some(vec![id.into()]));
    assert!(unsafe { registry.notify("missing", &[]) }.is_err());
    assert!(unsafe { registry.call_automation("missing", "test", &[]) }.is_err());
    assert!(unsafe { registry.activate("missing") }.is_err());
}

fn verify_rejected_libraries(
    registry: &mut NativeAddInRegistry,
    compiler: &compiler::FixtureCompiler,
) {
    for (name, flags) in [
        ("empty", vec!["empty_id"]),
        ("null", vec!["null_id"]),
        ("missing", vec!["missing_notify"]),
    ] {
        let path = rust_library(compiler, name, &flags);
        assert!(unsafe { registry.load(&path) }.is_err());
    }
    let newer = compiler.rust_library(
        "newer",
        &[],
        oblikovati_api::API_MAJOR,
        oblikovati_api::API_MINOR + 1,
    );
    assert!(unsafe { registry.load(&newer) }.is_err());
    assert!(unsafe { registry.load(Path::new("/missing/oblikovati-test-library.so")) }.is_err());
}
