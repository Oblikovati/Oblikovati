// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::sync::{Mutex, mpsc};
#[test]
fn runtime_keeps_failed_cleanup_available_until_retry_and_then_joins() {
    if isolation::case(
        "OBK_RUNTIME_CASE",
        "failures::runtime_keeps_failed_cleanup_available_until_retry_and_then_joins",
    ) {
        return;
    }
    let compiler = fixture_compiler::FixtureCompiler::new();
    let mut runtime = controls::new_runtime();
    let id = load_case(
        &mut runtime,
        &compiler,
        "retry",
        &["lifecycle", "deactivate_once_error"],
    );
    verify_cleanup_retry(&mut runtime, &id);
}
fn verify_cleanup_retry(runtime: &mut NativeAddInRuntime<NativeWakeCount>, id: &str) {
    unsafe { runtime.start_activate(id) }.unwrap();
    runtime.wait(&controls::RuntimeEcho).unwrap();
    unsafe { runtime.start_shutdown() }.unwrap();
    let failure = runtime.wait(&controls::RuntimeEcho).unwrap_err();
    assert!(failure.0.contains(id));
    assert_eq!(
        runtime.lifecycle_state(id).unwrap(),
        AddInState::NeedsCleanup
    );
    unsafe { runtime.start_shutdown() }.unwrap();
    assert_eq!(
        runtime.wait(&controls::RuntimeEcho).unwrap(),
        NativeCompletion::Shutdown
    );
    assert_eq!(runtime.lifecycle_state(id).unwrap(), AddInState::Inactive);
}

fn load_case<W: DispatchWakeup + 'static>(
    runtime: &mut NativeAddInRuntime<W>,
    compiler: &fixture_compiler::FixtureCompiler,
    name: &str,
    flags: &[&str],
) -> String {
    let path = compiler.rust_library(
        name,
        flags,
        oblikovati_api::API_MAJOR,
        oblikovati_api::API_MINOR,
    );
    unsafe { runtime.start_load(&path) }.unwrap();
    let NativeCompletion::Loaded(id) = runtime.wait(&controls::RuntimeEcho).unwrap() else {
        panic!("expected fixture identity");
    };
    id
}
struct PanickingModelHost;
impl Caller for PanickingModelHost {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        panic!("injected owner failure")
    }
}
#[test]
fn owner_panic_declines_foreign_activation_without_unwinding_c() {
    if isolation::case(
        "OBK_RUNTIME_CASE",
        "failures::owner_panic_declines_foreign_activation_without_unwinding_c",
    ) {
        return;
    }
    let compiler = fixture_compiler::FixtureCompiler::new();
    let mut runtime = controls::new_runtime();
    let id = load_case(
        &mut runtime,
        &compiler,
        "owner-panic",
        &["lifecycle", "callback_echo"],
    );
    verify_panic_cleanup(&mut runtime, &id);
}
fn verify_panic_cleanup(runtime: &mut NativeAddInRuntime<NativeWakeCount>, id: &str) {
    unsafe { runtime.start_activate(id) }.unwrap();
    assert!(runtime.wait(&PanickingModelHost).is_err());
    assert_eq!(
        runtime.lifecycle_state(id).unwrap(),
        AddInState::NeedsCleanup
    );
    unsafe { runtime.start_shutdown() }.unwrap();
    assert_eq!(
        runtime.wait(&controls::RuntimeEcho).unwrap(),
        NativeCompletion::Shutdown
    );
}

struct FailedNativeWake {
    entered: mpsc::Sender<()>,
    release: Mutex<mpsc::Receiver<()>>,
}
impl DispatchWakeup for FailedNativeWake {
    fn wake(&self) -> Result<(), String> {
        self.entered.send(()).unwrap();
        self.release.lock().unwrap().recv().unwrap();
        panic!("injected native wake failure")
    }
}
#[test]
fn unexpected_worker_loss_releases_pending_control_and_closes_owner_admission() {
    if isolation::case(
        "OBK_RUNTIME_CASE",
        "failures::unexpected_worker_loss_releases_pending_control_and_closes_owner_admission",
    ) {
        return;
    }
    let compiler = fixture_compiler::FixtureCompiler::new();
    let (mut runtime, ready, release) = failed_wake_runtime();
    let id = load_case(&mut runtime, &compiler, "worker-loss", &[]);
    ready.recv_timeout(Duration::from_secs(2)).unwrap();
    unsafe { runtime.start_activate(&id) }.unwrap();
    release.send(()).unwrap();
    let failure = runtime.wait(&controls::RuntimeEcho).unwrap_err();
    assert!(failure.0.contains("disconnected"));
    assert!(unsafe { runtime.start_shutdown() }.is_err());
    controls::verify_retired_transport(&runtime);
}
fn failed_wake_runtime() -> (
    NativeAddInRuntime<FailedNativeWake>,
    mpsc::Receiver<()>,
    mpsc::Sender<()>,
) {
    let (entered, ready) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let wakeup = FailedNativeWake {
        entered,
        release: Mutex::new(gate),
    };
    let runtime = NativeAddInRuntime::new(2, Duration::from_secs(2), wakeup).unwrap();
    (runtime, ready, release)
}
