// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::dispatch::{DispatchWakeup, owner_dispatch};
use std::{
    sync::mpsc::{self, Sender},
    thread,
    time::Duration,
};
struct QueueWakeSignal(Sender<()>);
impl DispatchWakeup for QueueWakeSignal {
    fn wake(&self) -> Result<(), String> {
        self.0.send(()).map_err(|error| error.to_string())
    }
}
#[test]
fn real_go_and_rust_activation_use_one_queued_callback_identity() {
    if isolated_case("queue::real_go_and_rust_activation_use_one_queued_callback_identity") {
        return;
    }
    let compiler = compiler::FixtureCompiler::new();
    let paths = activation_libraries(&compiler);
    let (signal, awake) = mpsc::channel();
    let (mut owner, caller) = owner_dispatch(2, Duration::from_secs(2), QueueWakeSignal(signal));
    let binding = HostCallbackBinding::install(Arc::new(caller)).unwrap();
    for path in paths {
        verify_activation(&path, &binding, &mut owner, &awake);
    }
    binding.close();
    owner.close();
}
fn activation_libraries(compiler: &compiler::FixtureCompiler) -> [std::path::PathBuf; 2] {
    let reference = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../../Oblikovati/head/internal/addinhost/testdata/echoaddin/main.go");
    [
        compiler.go_library(&reference),
        compiler.rust_library(
            "callback",
            &["lifecycle", "callback_echo"],
            oblikovati_api::API_MAJOR,
            oblikovati_api::API_MINOR,
        ),
    ]
}
fn verify_activation(
    path: &std::path::Path,
    binding: &HostCallbackBinding,
    owner: &mut oblikovati_automation::dispatch::OwnerDispatch<QueueWakeSignal>,
    awake: &mpsc::Receiver<()>,
) {
    // SAFETY: fixed ABI fixtures support this worker and matching allocator lifetimes.
    let mut addin = unsafe { oblikovati_addin_host::open_addin(path) }.unwrap();
    let (call, free) = binding.callbacks();
    let activation = thread::spawn(move || {
        unsafe { addin.activate(call, free) }.unwrap();
        addin
    });
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_eq!(binding.in_flight(), 1);
    assert_eq!(owner.drain(1, &EchoCaller).unwrap(), 1);
    let mut addin = activation.join().unwrap();
    assert_eq!(binding.in_flight(), 0);
    assert_eq!(binding.buffer_counts().outstanding, 0);
    unsafe { addin.deactivate() }.unwrap();
}
#[test]
fn accepted_native_timeout_preserves_copied_request_for_later_owner_execution() {
    if isolated_case(
        "queue::accepted_native_timeout_preserves_copied_request_for_later_owner_execution",
    ) {
        return;
    }
    let (signal, awake) = mpsc::channel();
    let (mut owner, caller) = owner_dispatch(1, Duration::from_millis(30), QueueWakeSignal(signal));
    let binding = HostCallbackBinding::install(Arc::new(caller)).unwrap();
    let (call, free) = binding.callbacks();
    let worker = thread::spawn(move || {
        let source = b"copied".to_vec();
        callback(call, free, c"late", &source)
    });
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    verify_late_result(worker, &binding, &mut owner);
}

#[derive(Default)]
struct RetainedCaller {
    request: std::cell::RefCell<Vec<u8>>,
}
impl Caller for RetainedCaller {
    fn call(&self, _: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        *self.request.borrow_mut() = request.to_vec();
        Ok(request.to_vec())
    }
}

fn verify_late_result(
    worker: thread::JoinHandle<(i32, Vec<u8>)>,
    binding: &HostCallbackBinding,
    owner: &mut oblikovati_automation::dispatch::OwnerDispatch<QueueWakeSignal>,
) {
    let (status, error) = worker.join().unwrap();
    assert_eq!(status, ffi::OBK_ERR);
    assert!(
        String::from_utf8(error)
            .unwrap()
            .contains("accepted request may still execute")
    );
    assert_eq!(binding.in_flight(), 0);
    let host = RetainedCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(host.request.borrow().as_slice(), b"copied");
    assert_eq!(owner.pending().unwrap(), 0);
    assert_eq!(binding.buffer_counts().outstanding, 0);
}
