// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use oblikovati_automation::dispatch::PollingDispatchWakeup;
use oblikovati_head::{
    DesktopDispatchOptions, DesktopObservations, DesktopOwner, DesktopScripts, HeadWork,
    ScriptDocumentOwner,
};
use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::file_access::NativeDocumentFiles;
use oblikovati_script::{IsolatedLuaEngine, ScriptLimits};
use oblikovati_session::files::FileWorkspace;
use std::{
    path::Path,
    time::{Duration, Instant},
};
type NativeScripts = DesktopScripts<
    DesktopOwner<
        DesktopObservations,
        DesktopObservations,
        NativeDocumentFiles,
        SystemGuidSource,
        DesktopObservations,
        PollingDispatchWakeup,
    >,
>;
fn desktop() -> NativeScripts {
    let observations = DesktopObservations::default();
    let files = FileWorkspace::new(observations.clone(), NativeDocumentFiles);
    let owner = DesktopOwner::new(
        files,
        SystemGuidSource,
        observations,
        "alpha".into(),
        PollingDispatchWakeup,
        DesktopDispatchOptions::default(),
    );
    DesktopScripts::new(owner, head_worker(), ScriptLimits::gui())
}
fn head_worker() -> IsolatedLuaEngine {
    IsolatedLuaEngine::new(
        Path::new("/usr/bin/env"),
        vec![
            "DISPLAY=:65530".into(),
            env!("CARGO_BIN_EXE_oblikovati-head").into(),
            oblikovati_script::WORKER_ARGUMENT.into(),
        ],
    )
}
fn finish<D: ScriptDocumentOwner>(desktop: &mut DesktopScripts<D>) {
    let deadline = Instant::now() + Duration::from_secs(3);
    while desktop.console().is_active() {
        assert!(
            Instant::now() < deadline,
            "expected bounded desktop script completion"
        );
        let activity = desktop.poll(32).unwrap();
        if let Some(wait) = activity.wake_after {
            std::thread::sleep(wait);
        }
    }
}
#[test]
fn head_private_worker_model_calls_stay_in_the_parent() {
    let mut desktop = desktop();
    desktop.console_mut().run("local part=oblikovati.documents.create{type='part',name='Lua desktop part'};print(part.id)").unwrap();
    assert!(!desktop.close_allowed().unwrap());
    finish(&mut desktop);
    assert!(
        desktop
            .console()
            .snapshot()
            .last_run
            .unwrap()
            .result
            .is_ok()
    );
    verify_parent_document_and_idle(&mut desktop);
}
fn verify_parent_document_and_idle(desktop: &mut NativeScripts) {
    assert_eq!(
        desktop
            .document_owner()
            .documents()
            .workspace()
            .documents()
            .len(),
        1
    );
    assert!(!desktop.close_allowed().unwrap());
    let idle = desktop.poll(32).unwrap();
    assert!(!idle.pending && idle.wake_after.is_none());
}
#[test]
fn check_cannot_change_documents_and_shutdown_retires_an_active_child() {
    let mut desktop = desktop();
    desktop
        .console_mut()
        .check("oblikovati.documents.create{type='part',name='not executed'}")
        .unwrap();
    finish(&mut desktop);
    assert!(
        desktop
            .document_owner()
            .documents()
            .workspace()
            .documents()
            .is_empty()
    );
    verify_shutdown_and_owner_return(desktop);
}
fn verify_shutdown_and_owner_return(mut desktop: NativeScripts) {
    desktop.console_mut().run("while true do end").unwrap();
    desktop.finish().unwrap();
    assert!(!desktop.console().is_active());
    assert!(
        desktop
            .console()
            .snapshot()
            .last_run
            .unwrap()
            .result
            .unwrap_err()
            .contains("shutdown")
    );
    let owner = desktop.into_document_owner();
    assert!(owner.documents().workspace().documents().is_empty());
}
#[test]
fn cancellation_preserves_committed_edits_and_requires_explicit_document_close() {
    let mut desktop = desktop();
    desktop
        .console_mut()
        .run(
            "oblikovati.documents.create{type='part',name='retained after stop'};while true do end",
        )
        .unwrap();
    wait_for_created_document(&mut desktop);
    assert!(desktop.poll(1).unwrap_err().contains("expected capacity"));
    desktop.console_mut().stop();
    finish(&mut desktop);
    verify_cancelled_ownership(desktop);
}
fn verify_cancelled_ownership(mut desktop: NativeScripts) {
    assert!(
        desktop
            .console()
            .snapshot()
            .last_run
            .unwrap()
            .result
            .unwrap_err()
            .contains("cancelled")
    );
    assert!(!desktop.close_allowed().unwrap());
    let files = desktop.into_document_owner().into_documents();
    assert_eq!(files.workspace().documents().len(), 1);
}
fn wait_for_created_document(desktop: &mut NativeScripts) {
    let deadline = Instant::now() + Duration::from_secs(2);
    while desktop
        .document_owner()
        .documents()
        .workspace()
        .documents()
        .is_empty()
    {
        assert!(Instant::now() < deadline);
        assert!(desktop.console().is_active());
        desktop.poll(32).unwrap();
        std::thread::yield_now();
    }
}

#[test]
fn direct_ui_host_access_remains_available_around_console_execution() {
    let mut desktop = desktop();
    let host = desktop.document_owner_mut().script_host();
    let reply = oblikovati_api::client::Client::new(host).documents().list();
    assert!(reply.is_ok());
    assert!(!desktop.console().is_active());
}

#[allow(dead_code)]
#[path = "../../addin-host/tests/support/resources.rs"]
mod resources;
#[allow(dead_code)]
#[path = "../../addin-host/tests/typed_workflow/compiler.rs"]
mod workflow_compiler;
#[test]
fn real_script_console_reads_the_native_catalog_and_model_owner() {
    let mut owner = desktop().into_document_owner();
    let directory = tempfile::tempdir().unwrap();
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../addin-host/tests/typed_workflow");
    let compiler = workflow_compiler::WorkflowCompiler::from_fixture_root(&root);
    let library = compiler.rust_workflow(&directory.path().join("script-catalog.opd"), true);
    let mut services = oblikovati_head::NativeAddInDesktop::new(
        &mut owner,
        Default::default(),
        PollingDispatchWakeup,
    )
    .unwrap();
    // This fixture is built from the pinned API-only source; its native lifecycle is trusted.
    unsafe { services.begin_startup(vec![library]) }.unwrap();
    run_catalog_console(services);
}
fn run_catalog_console<D: ScriptDocumentOwner>(services: D) {
    let mut methods = oblikovati_automation::file_host_methods();
    methods.extend([
        oblikovati_api::wire::METHOD_ADD_INS_LIST.into(),
        oblikovati_api::wire::METHOD_ADD_INS_GET.into(),
    ]);
    let mut scripts =
        DesktopScripts::with_methods(services, head_worker(), ScriptLimits::gui(), methods);
    await_native_startup(&mut scripts);
    scripts.console_mut().run("local entries=oblikovati.addins.list().addIns;assert(#entries==1);local found=oblikovati.addins.get{id=entries[1].id};assert(found.id==entries[1].id);assert(found.activated);assert(#oblikovati.documents.list().documents==1)").unwrap();
    finish(&mut scripts);
    let completed = scripts.console().snapshot().last_run.unwrap();
    assert!(completed.result.is_ok(), "{:?}", completed.result);
    scripts.finish().unwrap();
}
fn await_native_startup<D: ScriptDocumentOwner>(scripts: &mut DesktopScripts<D>) {
    let deadline = Instant::now() + Duration::from_secs(5);
    while !scripts.startup_ready() {
        assert!(
            Instant::now() < deadline,
            "native console startup: expected readiness within five seconds"
        );
        scripts.poll(32).unwrap();
        std::thread::yield_now();
    }
}

#[test]
fn desktop_console_reads_the_owned_material_catalog_without_a_document() {
    let mut desktop = desktop();
    desktop.console_mut().run("assert(#oblikovati.materials.list().materials==126);local p=oblikovati.materials.get{id='steel'};assert(p.isotropyClass=='isotropic');local a=oblikovati.appearances.get{id=p.appearanceId};assert(a.id==p.appearanceId);assert(#oblikovati.documents.list().documents==0)").unwrap();
    finish(&mut desktop);
    let result = desktop.console().snapshot().last_run.unwrap().result;
    assert!(result.is_ok(), "{result:?}");
    desktop.finish().unwrap();
}
