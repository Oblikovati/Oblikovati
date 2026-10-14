// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadWork, run_window_with_documents};
use oblikovati_viewport::ViewportCapture;
use std::path::PathBuf;
struct StartupWindowProbe<'a> {
    native: &'a mut NativeHeadWindow,
    capture: Option<ViewportCapture>,
}
impl HeadWindow for StartupWindowProbe<'_> {
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }

    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(update)
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        self.native.events(limit)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.capture = Some(super::scene_tests::capture_native_scene(self.native));
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
fn startup_scene_paths(directory: &std::path::Path) -> [PathBuf; 2] {
    let paths = [
        directory.join("first part.opd"),
        directory.join("second.opd"),
    ];
    for path in &paths {
        std::fs::write(path, super::scene_tests::native_scene_package()).unwrap();
    }
    paths
}
#[test]
fn startup_documents_are_open_visible_and_rendered_by_the_normal_owner_loop() {
    let directory = tempfile::tempdir().unwrap();
    let paths = startup_scene_paths(directory.path());
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_script_owner(native.wake.clone()).unwrap();
    let mut probe = startup_window_probe(&mut native);
    let options = HeadOptions { frames: Some(1) };
    let summary = run_window_with_documents(&mut probe, options, &mut owner, &paths).unwrap();
    verify_started_frame(summary, probe);
    verify_started_documents(&owner, &paths);
    verify_startup_sources(&paths);
}
fn startup_window_probe(native: &mut NativeHeadWindow) -> StartupWindowProbe<'_> {
    StartupWindowProbe {
        native,
        capture: None,
    }
}
fn verify_started_frame(summary: crate::HeadSummary, probe: StartupWindowProbe<'_>) {
    assert_eq!(summary.completed, 1);
    let capture = probe.capture.unwrap();
    assert!(super::scene_tests::solid_pixels(&capture) > 0);
    assert_eq!(probe.native.scene_status().unwrap().visible_bodies, 1);
    assert!(probe.native.view.camera_state().is_err());
}
fn verify_startup_sources(paths: &[PathBuf]) {
    for path in paths {
        assert_eq!(
            std::fs::read(path).unwrap(),
            super::scene_tests::native_scene_package()
        );
    }
}

fn verify_started_documents(owner: &NativeScriptDesktop, paths: &[PathBuf]) {
    let documents = owner.document_owner().documents();
    assert_eq!(documents.workspace().documents().len(), 2);
    let active = documents.workspace().active_document().unwrap();
    assert_eq!(documents.path(active).unwrap(), paths[1]);
    let caller = owner.document_owner().caller();
    assert!(
        oblikovati_api::client::Client::new(caller)
            .documents()
            .list()
            .is_err()
    );
}
#[test]
fn missing_startup_files_retire_the_native_target_and_owner_admission() {
    let directory = tempfile::tempdir().unwrap();
    let missing = directory.path().join("missing.opd");
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_script_owner(native.wake.clone()).unwrap();
    let options = HeadOptions { frames: Some(1) };
    let error =
        run_window_with_documents(&mut native, options, &mut owner, &[missing]).unwrap_err();
    assert!(error.contains("missing.opd"));
    assert!(native.view.camera_state().is_err());
    let documents = owner.document_owner().documents().workspace().documents();
    assert!(documents.is_empty());
    owner.finish().unwrap();
}
