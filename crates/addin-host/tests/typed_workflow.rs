// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use oblikovati_addin_host::HostCallbackBinding;
use oblikovati_api::{
    Error,
    client::{Caller, Client},
    ffi,
    wire::*,
};
use oblikovati_automation::{
    DocumentObserverFactory, FileHost,
    dispatch::{DispatchWakeup, OwnerDispatch, owner_dispatch},
};
use oblikovati_persistence::file_access::NativeDocumentFiles;
use oblikovati_session::files::FileWorkspace;
use std::{
    path::Path,
    sync::{
        Arc,
        mpsc::{self, Sender},
    },
    thread,
    time::Duration,
};
#[path = "typed_workflow/observations.rs"]
mod observations;
use observations::{WorkflowFactory, WorkflowObserver};
#[path = "typed_workflow/transport.rs"]
mod transport;
use transport::NativeCallbackCaller;
#[path = "typed_workflow/assertions.rs"]
mod assertions;
#[path = "typed_workflow/compiler.rs"]
mod compiler;
#[path = "typed_workflow/report.rs"]
mod report;
#[path = "typed_workflow/client.rs"]
mod workflow;
use assertions::{WorkflowFiles, assert_saved_workflow};
use workflow::WorkflowReport;
#[path = "typed_workflow/identity.rs"]
mod identity;
use identity::WorkflowGuids;
struct WorkflowWake(Sender<()>);
impl DispatchWakeup for WorkflowWake {
    fn wake(&self) -> Result<(), String> {
        self.0.send(()).map_err(|error| error.to_string())
    }
}

#[test]
fn typed_client_creates_constrained_part_edits_undoes_and_reopens_via_native_callbacks() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("native-part.opd");
    let (signal, awake) = mpsc::channel();
    let (mut owner, queued) = owner_dispatch(8, Duration::from_secs(5), WorkflowWake(signal));
    let binding = HostCallbackBinding::install(Arc::new(queued)).unwrap();
    let transport = NativeCallbackCaller::new(binding.callbacks());
    let worker = start_workflow(transport, &path);
    let mut runtime = WorkflowOwner::new();
    runtime.drive(&worker, &awake, &mut owner);
    let report = worker.join().unwrap();
    assert_saved_workflow(&runtime.files, &path, &report);
    verify_loaded_workflow(&binding, &awake, &mut owner, &mut runtime, directory.path());
    assert_eq!(binding.buffer_counts().outstanding, 0);
    binding.close();
    owner.close();
}

fn start_workflow(
    transport: NativeCallbackCaller,
    path: &Path,
) -> thread::JoinHandle<workflow::WorkflowReport> {
    let path = path.to_owned();
    thread::spawn(move || workflow::execute(Client::new(transport), &path))
}

struct WorkflowOwner {
    files: WorkflowFiles,
    guids: WorkflowGuids,
    factory: WorkflowFactory,
}
impl WorkflowOwner {
    fn new() -> Self {
        Self {
            files: WorkflowFiles::new(WorkflowObserver::default(), NativeDocumentFiles),
            guids: WorkflowGuids(0),
            factory: WorkflowFactory,
        }
    }
    fn drive<T>(
        &mut self,
        worker: &thread::JoinHandle<T>,
        awake: &mpsc::Receiver<()>,
        owner: &mut OwnerDispatch<WorkflowWake>,
    ) {
        let deadline = std::time::Instant::now() + Duration::from_secs(10);
        while !worker.is_finished() {
            assert!(
                std::time::Instant::now() < deadline,
                "expected typed workflow completion before deadline"
            );
            self.drain_woken(awake, owner);
        }
    }
    fn drain_woken(&mut self, awake: &mpsc::Receiver<()>, owner: &mut OwnerDispatch<WorkflowWake>) {
        if awake.recv_timeout(Duration::from_millis(20)).is_err() {
            return;
        }
        let host = FileHost::new(&mut self.files, &mut self.guids, "test")
            .with_observer_factory(&mut self.factory);
        assert!(owner.drain(8, &host).unwrap() > 0);
    }
}

fn verify_loaded_workflow(
    binding: &HostCallbackBinding,
    awake: &mpsc::Receiver<()>,
    owner: &mut OwnerDispatch<WorkflowWake>,
    runtime: &mut WorkflowOwner,
    directory: &Path,
) {
    let compiler = compiler::WorkflowCompiler::new();
    let mut radius_bits = None;
    for language in ["rust", "go"] {
        let path = directory.join(format!("{language}-addin.opd"));
        let library = match language {
            "rust" => compiler.rust_workflow(&path, false),
            _ => compiler.go_workflow(&path, false),
        };
        let report = verify_loaded_library(binding, awake, owner, runtime, &library, &path);
        assert_eq!(
            *radius_bits.get_or_insert(report.radius_bits),
            report.radius_bits
        );
    }
}
fn verify_loaded_library(
    binding: &HostCallbackBinding,
    awake: &mpsc::Receiver<()>,
    owner: &mut OwnerDispatch<WorkflowWake>,
    runtime: &mut WorkflowOwner,
    library: &Path,
    path: &Path,
) -> workflow::WorkflowReport {
    // SAFETY: API-only fixture uses the same canonical prototypes and supports this worker affinity.
    let mut addin = unsafe { oblikovati_addin_host::open_addin(library) }.unwrap();
    let (call, free) = binding.callbacks();
    let activation = thread::spawn(move || {
        unsafe { addin.activate(call, free) }.unwrap();
        addin
    });
    runtime.drive(&activation, awake, owner);
    let mut addin = activation.join().unwrap();
    let report = loaded_report(&mut addin);
    assert_saved_workflow(&runtime.files, path, &report);
    unsafe { addin.deactivate() }.unwrap();
    report
}
fn loaded_report(addin: &mut oblikovati_addin_host::LoadedAddIn) -> workflow::WorkflowReport {
    assert_eq!(addin.metadata().major, oblikovati_api::API_MAJOR);
    assert_eq!(addin.metadata().minor, oblikovati_api::API_MINOR);
    // SAFETY: fixture notification reads no payload and makes no host calls.
    unsafe { addin.notify(b"{}") }.unwrap();
    // SAFETY: fixture reports only its copied workflow observations; returned bytes use its matching ObkFree.
    let bytes = unsafe { addin.automation("report", &[]) }.unwrap();
    serde_json::from_slice(&bytes).unwrap()
}

#[path = "support/resources.rs"]
mod resources;
