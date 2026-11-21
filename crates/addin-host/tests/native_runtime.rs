// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use oblikovati_addin_host::{AddInState, NativeAddInRuntime, NativeCompletion};
use oblikovati_api::{Error, client::Caller, contract::AddInRegistry, types::*, wire::*};
use oblikovati_automation::{DocumentObserverFactory, FileHost, dispatch::DispatchWakeup};
use oblikovati_persistence::file_access::NativeDocumentFiles;
use oblikovati_session::files::FileWorkspace;
use std::{
    path::{Path, PathBuf},
    sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    },
    time::Duration,
};
#[path = "support/compiler.rs"]
mod fixture_compiler;
#[path = "support/isolation.rs"]
mod isolation;
#[path = "typed_workflow/observations.rs"]
mod observations;
#[path = "typed_workflow/compiler.rs"]
mod workflow_compiler;
use observations::{WorkflowFactory, WorkflowObserver};
#[path = "typed_workflow/report.rs"]
mod report;
use report::WorkflowReport;
#[path = "typed_workflow/assertions.rs"]
mod assertions;
use assertions::{WorkflowFiles, assert_saved_workflow};
#[path = "typed_workflow/identity.rs"]
mod identity;
use identity::WorkflowGuids;
#[path = "native_runtime/controls.rs"]
mod controls;
#[derive(Clone)]
struct NativeWakeCount(Arc<AtomicUsize>);
impl DispatchWakeup for NativeWakeCount {
    fn wake(&self) -> Result<(), String> {
        self.0.fetch_add(1, Ordering::SeqCst);
        Ok(())
    }
}
struct DocumentRuntimeOwner {
    files: WorkflowFiles,
    guids: WorkflowGuids,
    factory: WorkflowFactory,
}
impl DocumentRuntimeOwner {
    fn new() -> Self {
        Self {
            files: WorkflowFiles::new(WorkflowObserver::default(), NativeDocumentFiles),
            guids: WorkflowGuids(0),
            factory: WorkflowFactory,
        }
    }
    fn wait<W: DispatchWakeup + 'static>(
        &mut self,
        runtime: &mut NativeAddInRuntime<W>,
    ) -> Result<NativeCompletion, Error> {
        let host = FileHost::new(&mut self.files, &mut self.guids, "alpha")
            .with_observer_factory(&mut self.factory);
        runtime.wait(&host)
    }
}
#[test]
fn owner_runtime_runs_loaded_go_and_rust_document_workflows_with_live_catalog_queries() {
    if isolation::case(
        "OBK_RUNTIME_CASE",
        "owner_runtime_runs_loaded_go_and_rust_document_workflows_with_live_catalog_queries",
    ) {
        return;
    }
    let wakes = NativeWakeCount(Arc::new(AtomicUsize::new(0)));
    let mut runtime = NativeAddInRuntime::new(8, Duration::from_secs(5), wakes.clone()).unwrap();
    let mut owner = DocumentRuntimeOwner::new();
    verify_initial_runtime(&mut runtime);
    run_document_libraries(&mut runtime, &mut owner);
    unsafe { runtime.start_shutdown() }.unwrap();
    assert_eq!(
        owner.wait(&mut runtime).unwrap(),
        NativeCompletion::Shutdown
    );
    assert_eq!(runtime.last_buffer_counts().outstanding, 0);
    assert!(wakes.0.load(Ordering::SeqCst) > 0);
}
fn run_document_libraries(
    runtime: &mut NativeAddInRuntime<NativeWakeCount>,
    owner: &mut DocumentRuntimeOwner,
) {
    let directory = tempfile::tempdir().unwrap();
    let compiler = workflow_compiler::WorkflowCompiler::new();
    let mut expected = None;
    for language in ["rust", "go"] {
        let path = directory.path().join(format!("{language}-runtime.opd"));
        let library = match language {
            "rust" => compiler.rust_workflow(&path, true),
            _ => compiler.go_workflow(&path, true),
        };
        let bits = verify_document_addin(runtime, owner, &library, &path);
        assert_eq!(*expected.get_or_insert(bits), bits);
    }
}

fn verify_initial_runtime(runtime: &mut NativeAddInRuntime<NativeWakeCount>) {
    assert_eq!(runtime.registered(), Some(Vec::new()));
    assert_eq!(runtime.last_buffer_counts().outstanding, 0);
    assert!(runtime.describe("missing").is_err());
    assert!(runtime.lifecycle_state("missing").is_err());
    assert!(!runtime.is_active("missing".into()));
    assert_eq!(runtime.load_behavior("missing".into()), LOAD_ON_STARTUP);
    assert!(runtime.wait(&controls::RuntimeEcho).is_err());
    assert!(runtime.poll(0, &controls::RuntimeEcho).unwrap().is_none());
}
fn verify_document_addin(
    runtime: &mut NativeAddInRuntime<NativeWakeCount>,
    owner: &mut DocumentRuntimeOwner,
    library: &Path,
    path: &Path,
) -> u64 {
    unsafe { runtime.start_load(library) }.unwrap();
    assert!(unsafe { runtime.start_load(library) }.is_err());
    let NativeCompletion::Loaded(id) = owner.wait(runtime).unwrap() else {
        panic!("expected native load identity");
    };
    runtime.start_policy(&id, LOAD_ON_DEMAND).unwrap();
    assert_eq!(owner.wait(runtime).unwrap(), NativeCompletion::Finished);
    assert_eq!(runtime.load_behavior(id.clone()), LOAD_ON_DEMAND);
    unsafe { runtime.start_activate(&id) }.unwrap();
    assert_eq!(owner.wait(runtime).unwrap(), NativeCompletion::Finished);
    assert!(runtime.is_active(id.clone()));
    let report = verify_runtime_report(runtime, owner, &id);
    assert_saved_workflow(&owner.files, path, &report);
    unsafe { runtime.start_deactivate(&id) }.unwrap();
    assert_eq!(owner.wait(runtime).unwrap(), NativeCompletion::Finished);
    assert_eq!(runtime.lifecycle_state(&id).unwrap(), AddInState::Inactive);
    report.radius_bits
}
fn verify_runtime_report(
    runtime: &mut NativeAddInRuntime<NativeWakeCount>,
    owner: &mut DocumentRuntimeOwner,
    id: &str,
) -> WorkflowReport {
    unsafe { runtime.start_notify(id, b"{}") }.unwrap();
    assert_eq!(owner.wait(runtime).unwrap(), NativeCompletion::Finished);
    unsafe { runtime.start_automation(id, "report", &[]) }.unwrap();
    let NativeCompletion::Automation(bytes) = owner.wait(runtime).unwrap() else {
        panic!("expected copied native report");
    };
    serde_json::from_slice(&bytes).unwrap()
}

#[path = "native_runtime/failures.rs"]
mod failures;

#[path = "support/resources.rs"]
mod resources;
