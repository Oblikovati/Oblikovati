// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_automation::dispatch::PollingDispatchWakeup;
use oblikovati_head::{
    DesktopDispatchOptions, DesktopObservations, DesktopOwner, DesktopScripts, HeadOptions,
    HeadPresentation, HeadSummary, HeadWindow, HeadWork, HeadWorkStatus, NativeHeadWindow,
    run_window_with_owner,
};
use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::file_access::NativeDocumentFiles;
use oblikovati_script::{IsolatedLuaEngine, ScriptLimits};
use oblikovati_session::files::FileWorkspace;
use oblikovati_viewport::ViewportEvent;
use std::{cell::Cell, path::Path, rc::Rc, time::Duration};
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
struct ScriptWindow {
    native: NativeHeadWindow,
    finished: Rc<Cell<bool>>,
    timed_waits: usize,
}
impl HeadWindow for ScriptWindow {
    fn submit_sketch_scene(
        &mut self,
        update: oblikovati_head::HeadSketchSceneUpdate,
    ) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }

    fn submit_scene(&mut self, update: oblikovati_head::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(update)
    }

    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        if self.finished.get() {
            events.push(ViewportEvent::CloseRequested);
        }
        Ok(events)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<Duration>) -> Result<bool, String> {
        if self.finished.get() {
            return Ok(false);
        }
        self.timed_waits += usize::from(timeout.is_some());
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
struct ScriptOwner {
    desktop: NativeScripts,
    finished: Rc<Cell<bool>>,
}
impl HeadWork for ScriptOwner {
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        let activity = self.desktop.poll(budget)?;
        if !self.desktop.console().is_active() {
            let outcome = self
                .desktop
                .console()
                .snapshot()
                .last_run
                .expect("expected completed script");
            outcome
                .result
                .map_err(|error| format!("native script fixture: {error}"))?;
            self.finished.set(true);
        }
        Ok(activity)
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        self.desktop.close_allowed()
    }
    fn finish(&mut self) -> Result<(), String> {
        self.desktop.finish()
    }
}
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
    let engine = IsolatedLuaEngine::new(
        Path::new(env!("CARGO_BIN_EXE_oblikovati-head")),
        vec![oblikovati_script::WORKER_ARGUMENT.into()],
    );
    DesktopScripts::new(owner, engine, ScriptLimits::gui())
}
#[test]
fn native_frames_service_lua_file_calls_through_the_same_parent_owner() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("script-parent.opd");
    let finished = Rc::new(Cell::new(false));
    let mut owner = ScriptOwner {
        desktop: desktop(),
        finished: finished.clone(),
    };
    start_native_script(&mut owner, &path);
    let mut window = ScriptWindow {
        native: NativeHeadWindow::open().unwrap(),
        finished,
        timed_waits: 0,
    };
    let summary = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    verify(&owner, &window, &path, summary);
}
fn start_native_script(owner: &mut ScriptOwner, path: &Path) {
    let source = source(path.to_str().unwrap());
    owner.desktop.console_mut().run(&source).unwrap();
}
fn source(path: &str) -> String {
    format!(
        "local doc=oblikovati.documents.create{{type='part',name='Script native part'}};oblikovati.documents.saveAs{{document=doc.id,newFullDocumentName={path:?}}};oblikovati.documents.close{{id=doc.id,force=false}};oblikovati.documents.open{{fullDocumentName={path:?}}};print('saved and reopened')"
    )
}
fn verify(owner: &ScriptOwner, window: &ScriptWindow, path: &Path, summary: HeadSummary) {
    let snapshot = owner.desktop.console().snapshot();
    assert!(snapshot.last_run.unwrap().result.is_ok());
    assert_eq!(snapshot.output, ["saved and reopened"]);
    let workspace = owner.desktop.document_owner().documents().workspace();
    assert_eq!(workspace.documents().len(), 1);
    assert!(!workspace.needs_save(workspace.documents()[0]).unwrap());
    assert!(path.is_file());
    assert!(summary.completed > 0 && window.timed_waits > 0);
}
