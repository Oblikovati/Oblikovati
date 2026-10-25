// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadWork, HeadWorkStatus, NativeAddInDesktop, ScriptDocumentOwner};
use oblikovati_addin_host::NativeCompletion;
use std::{cell::Cell, collections::VecDeque, path::PathBuf, rc::Rc};
#[allow(dead_code)]
#[path = "../../addin-host/tests/typed_workflow/compiler.rs"]
mod compiler;
#[path = "native_services/window.rs"]
mod controlled_window;
#[path = "../../addin-host/tests/support/isolation.rs"]
mod isolation;
#[allow(dead_code)]
#[path = "../../addin-host/tests/support/resources.rs"]
mod resources;
use controlled_window::CloseWhenReadyWindow;
#[derive(Clone, Copy, Debug)]
enum StartupStep {
    Loading,
    Activating,
    Done,
}
type TestNativeServices<'a> = NativeAddInDesktop<'a, NativeDesktopOwner, NativeOwnerWake>;
struct FixtureStartup<'a> {
    services: TestNativeServices<'a>,
    libraries: VecDeque<PathBuf>,
    step: StartupStep,
    done: Rc<Cell<bool>>,
}
impl FixtureStartup<'_> {
    fn next_library(&mut self) -> Result<(), String> {
        let Some(path) = self.libraries.pop_front() else {
            self.step = StartupStep::Done;
            self.done.set(true);
            return Ok(());
        };
        self.step = StartupStep::Loading;
        unsafe { self.services.start_load(&path) }
    }
    fn completion(&mut self, outcome: NativeCompletion) -> Result<(), String> {
        match (self.step, outcome) {
            (StartupStep::Loading, NativeCompletion::Loaded(id)) => {
                verify_native_catalog(&mut self.services, &id);
                self.step = StartupStep::Activating;
                unsafe { self.services.start_activate(&id) }
            }
            (StartupStep::Activating, NativeCompletion::Finished) => self.next_library(),
            (step, outcome) => Err(format!(
                "fixture startup {step:?}, {outcome:?}: expected load then activation"
            )),
        }
    }
}
impl HeadWork for FixtureStartup<'_> {
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        let activity = self.services.poll(budget)?;
        if let Some(outcome) = self.services.take_native_completion() {
            self.completion(outcome?)?;
        }
        Ok(activity)
    }
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        self.services.take_scene()
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        self.services.close_allowed()
    }
    fn finish(&mut self) -> Result<(), String> {
        self.services.finish()
    }
}
#[test]
fn native_frame_owner_services_real_go_and_rust_addins() {
    let case = "native::addin_services_tests::native_frame_owner_services_real_go_and_rust_addins";
    if isolation::case("OBLIKOVATI_HEAD_NATIVE_SERVICES", case) {
        return;
    }
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let done = Rc::new(Cell::new(false));
    let compiler = compiler::WorkflowCompiler::from_fixture_root(
        &PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../addin-host/tests/typed_workflow"),
    );
    let directory = tempfile::tempdir().unwrap();
    run_native_fixture_frame(&mut window, &mut owner, done, &compiler, directory.path());
    assert_eq!(owner.documents().workspace().documents().len(), 2);
    assert!(directory.path().join("rust.opd").is_file());
    assert!(directory.path().join("go.opd").is_file());
    verify_reopened_documents(&owner);
}
fn run_native_fixture_frame(
    window: &mut NativeHeadWindow,
    owner: &mut NativeDesktopOwner,
    done: Rc<Cell<bool>>,
    compiler: &compiler::WorkflowCompiler,
    directory: &std::path::Path,
) {
    let libraries = compiled_libraries(compiler, directory);
    let services = NativeAddInDesktop::new(
        owner,
        Default::default(),
        NativeOwnerWake(window.wake.clone()),
    )
    .unwrap();
    let mut startup = FixtureStartup {
        services,
        libraries,
        step: StartupStep::Loading,
        done,
    };
    assert!(startup.services.poll(1).is_err());
    startup.next_library().unwrap();
    assert!(!startup.services.close_allowed().unwrap());
    run_fixture_loop(window, &mut startup);
    assert!(unsafe { startup.services.start_activate("retired") }.is_err());
    startup.services.finish().unwrap();
    assert!(startup.services.poll(32).is_err());
}
fn compiled_libraries(
    compiler: &compiler::WorkflowCompiler,
    directory: &std::path::Path,
) -> VecDeque<PathBuf> {
    VecDeque::from([
        compiler.rust_workflow(&directory.join("rust.opd"), true),
        compiler.go_workflow(&directory.join("go.opd"), true),
    ])
}
fn run_fixture_loop(window: &mut NativeHeadWindow, startup: &mut FixtureStartup<'_>) {
    let mut display = CloseWhenReadyWindow {
        window,
        done: startup.done.clone(),
        deadline: std::time::Instant::now() + std::time::Duration::from_secs(20),
    };
    let summary =
        crate::run_window_with_owner(&mut display, HeadOptions::default(), startup).unwrap();
    assert!(summary.completed > 0);
}

fn verify_reopened_documents(owner: &NativeDesktopOwner) {
    let workspace = owner.documents().workspace();
    let mut expected = None;
    let tolerance = oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance;
    for id in workspace.documents() {
        let snapshot = workspace.session(*id).unwrap().snapshot();
        let sketch = &snapshot.sketches.sketches()[0];
        snapshot.require_resolved_sketch(sketch.id()).unwrap();
        let radius = sketch.geometry().circles()[0].radius;
        assert!((radius - 3.5).abs() / 3.5 <= tolerance);
        assert_eq!(*expected.get_or_insert(radius.to_bits()), radius.to_bits());
        assert!(snapshot.parameters.find("width").is_some());
        assert!(!workspace.needs_save(*id).unwrap());
    }
}
#[test]
fn native_services_finish_failed_pending_load_without_losing_owner() {
    let case = "native::addin_services_tests::native_services_finish_failed_pending_load_without_losing_owner";
    if isolation::case("OBLIKOVATI_HEAD_NATIVE_SERVICES", case) {
        return;
    }
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    verify_pending_load_retirement(&mut owner, NativeOwnerWake(window.wake.clone()));
    assert!(owner.documents().workspace().documents().is_empty());
    window.finish().unwrap();
}
fn verify_pending_load_retirement(owner: &mut NativeDesktopOwner, wake: NativeOwnerWake) {
    let directory = tempfile::tempdir().unwrap();
    let missing = directory.path().join("missing.so");
    let mut services = NativeAddInDesktop::new(owner, Default::default(), wake).unwrap();
    verify_service_delegation(&mut services);
    unsafe { services.start_load(&missing) }.unwrap();
    assert!(!services.close_allowed().unwrap());
    assert!(services.finish().unwrap_err().contains("missing.so"));
    assert!(unsafe { services.start_load(&missing) }.is_err());
    services.finish().unwrap();
}

fn verify_service_delegation(services: &mut TestNativeServices<'_>) {
    assert!(services.sketch_preview().is_none());
    assert!(services.sketch_edit().is_none());
    services
        .decline_sketch_edit("test camera refusal".into())
        .unwrap();
    assert!(
        services
            .take_command_notice()
            .unwrap()
            .message()
            .contains("test camera refusal")
    );
    let zero_document = services.gui_command_state().unwrap();
    assert!(zero_document.documents.is_empty());
    assert_eq!(zero_document.ribbon_key, "ZeroDoc");
    assert_eq!(zero_document.ribbon_tabs[0].name, "Get Started");
    assert!(
        !services
            .process_window_inputs(&[], &crate::placement_test_inputs::ProjectionlessWindow)
            .unwrap()
    );
    assert!(!services.complete_save_path(0, None).unwrap());
    let save = ViewportEvent::Input(oblikovati_viewport::ViewportInput::Key {
        key: oblikovati_viewport::ViewportKey::Save,
        pressed: true,
    });
    assert!(services.process_inputs(&[save]).unwrap());
    assert_eq!(
        services.take_command_notice(),
        Some(crate::DesktopCommandNotice::NoActiveDocument)
    );
    let documents = oblikovati_api::client::Client::new(services.script_host())
        .documents()
        .list()
        .unwrap();
    assert!(documents.documents.unwrap().is_empty());
}

fn verify_native_catalog(services: &mut TestNativeServices<'_>, id: &str) {
    let client = oblikovati_api::client::Client::new(services.script_host());
    let entries = client.add_ins().list().unwrap().add_ins.unwrap();
    let listed = entries.iter().find(|entry| entry.id == id).unwrap();
    let described = client
        .add_ins()
        .get(&oblikovati_api::wire::AddInRefArgs { id: id.into() })
        .unwrap();
    assert_eq!(*listed, described);
    assert!(!described.activated);
}
