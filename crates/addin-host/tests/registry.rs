// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use oblikovati_addin_host::{AddInState, HostCallbackBinding, NativeAddInRegistry};
use oblikovati_api::{Error, client::Caller, contract::AddInRegistry, ffi, types::*};
use oblikovati_automation::dispatch::{DispatchWakeup, OwnerDispatch, owner_dispatch};
use std::{
    path::{Path, PathBuf},
    process::Command,
    sync::{
        Arc,
        mpsc::{self, Receiver, Sender},
    },
    thread,
    time::Duration,
};
#[path = "registry/checks.rs"]
mod checks;
#[path = "support/compiler.rs"]
mod compiler;
#[path = "registry/shutdown.rs"]
mod shutdown;
struct RegistryWake(Sender<()>);
impl DispatchWakeup for RegistryWake {
    fn wake(&self) -> Result<(), String> {
        self.0.send(()).map_err(|error| error.to_string())
    }
}
struct RegistryEcho;
impl Caller for RegistryEcho {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        assert_eq!(method, "echo");
        Ok(request.to_vec())
    }
}
struct RegistryHarness {
    registry: NativeAddInRegistry,
    owner: OwnerDispatch<RegistryWake>,
    awake: Receiver<()>,
    callbacks: (ffi::ObkHostCall, ffi::ObkHostFree),
}
impl RegistryHarness {
    fn new() -> Self {
        let (signal, awake) = mpsc::channel();
        let (owner, queued) = owner_dispatch(8, Duration::from_secs(2), RegistryWake(signal));
        let binding = HostCallbackBinding::install(Arc::new(queued)).unwrap();
        let callbacks = binding.callbacks();
        Self {
            registry: NativeAddInRegistry::new(binding),
            owner,
            awake,
            callbacks,
        }
    }
}
fn activate_with_owner(
    mut registry: NativeAddInRegistry,
    owner: &mut OwnerDispatch<RegistryWake>,
    awake: &Receiver<()>,
    id: &str,
) -> NativeAddInRegistry {
    let id = id.to_owned();
    let worker = thread::spawn(move || {
        unsafe { registry.activate(&id) }.unwrap();
        unsafe { registry.activate(&id) }.unwrap();
        registry
    });
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_eq!(owner.drain(1, &RegistryEcho).unwrap(), 1);
    let registry = worker.join().unwrap();
    assert_eq!(owner.pending().unwrap(), 0);
    registry
}

fn isolated(test: &str) -> bool {
    if std::env::var("OBK_REGISTRY_CASE").as_deref() == Ok(test) {
        return false;
    }
    let output = Command::new(std::env::current_exe().unwrap())
        .args(["--exact", test, "--nocapture"])
        .env("OBK_REGISTRY_CASE", test)
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "registry child {test}: {} {}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    true
}
fn rust_library(compiler: &compiler::FixtureCompiler, name: &str, flags: &[&str]) -> PathBuf {
    compiler.rust_library(
        name,
        flags,
        oblikovati_api::API_MAJOR,
        oblikovati_api::API_MINOR,
    )
}
fn go_library(compiler: &compiler::FixtureCompiler) -> PathBuf {
    let reference = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../../Oblikovati/head/internal/addinhost/testdata/echoaddin/main.go");
    compiler.go_library(&reference)
}
fn load(registry: &mut NativeAddInRegistry, path: &Path) -> String {
    // SAFETY: fixtures use canonical exports and resident mappings, with no external lifecycle aliases.
    unsafe { registry.load(path) }.unwrap()
}

fn finish_harness(harness: &mut RegistryHarness) {
    unsafe { harness.registry.shutdown() }.unwrap();
    harness.owner.close();
}
fn assert_lifecycle(registry: &NativeAddInRegistry, id: &str, expected: AddInState) {
    assert_eq!(registry.lifecycle_state(id).unwrap(), expected);
}
