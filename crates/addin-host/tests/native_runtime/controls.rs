// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) struct RuntimeEcho;
impl Caller for RuntimeEcho {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        assert_eq!(method, "echo");
        Ok(request.to_vec())
    }
}
#[test]
fn persistent_worker_keeps_foreign_affinity_and_drains_activation_and_cleanup() {
    if isolation::case(
        "OBK_RUNTIME_CASE",
        "controls::persistent_worker_keeps_foreign_affinity_and_drains_activation_and_cleanup",
    ) {
        return;
    }
    let compiler = fixture_compiler::FixtureCompiler::new();
    let mut runtime = new_runtime();
    for path in affinity_libraries(&compiler) {
        verify_affinity_library(&mut runtime, &path);
    }
    finish_runtime(&mut runtime);
}
fn finish_runtime(runtime: &mut NativeAddInRuntime<NativeWakeCount>) {
    unsafe { runtime.start_shutdown() }.unwrap();
    assert_eq!(
        runtime.wait(&RuntimeEcho).unwrap(),
        NativeCompletion::Shutdown
    );
    assert!(unsafe { runtime.start_shutdown() }.is_err());
    assert!(runtime.poll(0, &RuntimeEcho).unwrap().is_none());
    verify_retired_transport(runtime);
}

pub(super) fn new_runtime() -> NativeAddInRuntime<NativeWakeCount> {
    NativeAddInRuntime::new(
        2,
        Duration::from_secs(2),
        NativeWakeCount(Arc::new(AtomicUsize::new(0))),
    )
    .unwrap()
}

fn affinity_libraries(compiler: &fixture_compiler::FixtureCompiler) -> [PathBuf; 2] {
    let reference = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../../Oblikovati/head/internal/addinhost/testdata/echoaddin/main.go");
    [
        compiler.rust_library(
            "affinity",
            &[
                "lifecycle",
                "callback_echo",
                "deactivate_echo",
                "automation",
                "check_affinity",
            ],
            oblikovati_api::API_MAJOR,
            oblikovati_api::API_MINOR,
        ),
        compiler.go_library(&reference),
    ]
}
fn verify_affinity_library(runtime: &mut NativeAddInRuntime<NativeWakeCount>, path: &Path) {
    unsafe { runtime.start_load(path) }.unwrap();
    let NativeCompletion::Loaded(id) = runtime.wait(&RuntimeEcho).unwrap() else {
        panic!("expected native load");
    };
    unsafe { runtime.start_activate(&id) }.unwrap();
    assert_eq!(
        runtime.wait(&RuntimeEcho).unwrap(),
        NativeCompletion::Finished
    );
    verify_native_exports(runtime, &id);
    if id == "rust-fixture" {
        verify_cleanup_failure(runtime, &id);
    }
}
fn verify_native_exports(runtime: &mut NativeAddInRuntime<NativeWakeCount>, id: &str) {
    unsafe { runtime.start_notify(id, b"{}") }.unwrap();
    assert_eq!(
        runtime.wait(&RuntimeEcho).unwrap(),
        NativeCompletion::Finished
    );
    unsafe { runtime.start_automation(id, "echo", b"ping") }.unwrap();
    assert!(matches!(
        runtime.wait(&RuntimeEcho).unwrap(),
        NativeCompletion::Automation(_)
    ));
}
fn verify_cleanup_failure(runtime: &mut NativeAddInRuntime<NativeWakeCount>, id: &str) {
    runtime.start_policy(id, LOAD_DISABLED).unwrap();
    assert_eq!(
        runtime.wait(&RuntimeEcho).unwrap(),
        NativeCompletion::Finished
    );
    runtime.start_policy(id, AddInLoadBehavior(255)).unwrap();
    assert!(runtime.wait(&RuntimeEcho).is_err());
    assert_eq!(runtime.load_behavior(id.into()), LOAD_DISABLED);
}

pub(super) fn verify_retired_transport<W: DispatchWakeup + 'static>(
    runtime: &NativeAddInRuntime<W>,
) {
    let transport = runtime.caller();
    let reply = std::thread::spawn(move || transport.call("echo", b"late"))
        .join()
        .unwrap();
    assert!(reply.unwrap_err().0.contains("closed"));
    assert_eq!(runtime.last_buffer_counts().outstanding, 0);
}
