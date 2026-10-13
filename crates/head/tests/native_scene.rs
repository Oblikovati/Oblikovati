// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWork;
use crate::scene_fixture as fixture;
use oblikovati_api::{client::Client, wire::*};
use oblikovati_viewport::ViewportCapture;
/// Encode the shared stored-prism fixture; example: `native_scene_package()` for live file-open tests.
pub(super) fn native_scene_package() -> Vec<u8> {
    let snapshot = fixture::selected_prism_document();
    let package = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap();
    let parameters = package
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap();
    let sketches = parameters
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap();
    sketches
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
        .encode()
        .unwrap()
}
/// Open a real stored part through the typed API; example: `open_native_part(owner, path)` for foreground acceptance.
pub(super) fn open_native_part(owner: &mut NativeDesktopOwner, path: &std::path::Path) -> u64 {
    std::fs::write(path, native_scene_package()).unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
/// Edit a retained part parameter; example: `native_set(owner, "width", "6 cm")`.
pub(super) fn native_set(owner: &mut NativeDesktopOwner, name: &str, expression: &str) {
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: name.into(),
            expression: expression.into(),
            ..Default::default()
        })
        .unwrap();
}
/// Publish the current owner result and verify its native title; example: `publish_native_scene(owner, window)`.
pub(super) fn publish_native_scene(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
) -> ViewportCapture {
    window
        .submit_scene(owner.take_scene().unwrap().unwrap())
        .unwrap();
    assert_eq!(
        window.view.native_title().unwrap(),
        window.scene_status().unwrap().title()
    );
    capture_native_scene(window)
}
/// Capture a completed scene and check its X11 pixels; example: `capture_native_scene(window)`.
pub(super) fn capture_native_scene(window: &mut NativeHeadWindow) -> ViewportCapture {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(3);
    loop {
        match window.view.present().unwrap() {
            ViewportPresentation::Completed(capture) => {
                verify_native_pixels(window, &capture, deadline);
                return capture;
            }
            ViewportPresentation::SurfaceChanged(_) => {
                assert!(std::time::Instant::now() < deadline);
                window.view.poll_events(64).unwrap();
            }
        }
    }
}
fn verify_native_pixels(
    window: &NativeHeadWindow,
    capture: &ViewportCapture,
    deadline: std::time::Instant,
) {
    while window.view.native_pixels().unwrap() != capture.rgba {
        assert!(std::time::Instant::now() < deadline);
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
}
/// Count the visible fixture body pixels; example: `solid_pixels(&capture)` after native presentation.
pub(super) fn solid_pixels(capture: &ViewportCapture) -> usize {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| **pixel == [0, 255, 0, 255])
        .count()
}
#[test]
fn native_head_displays_owned_edits_undo_failures_and_close_without_old_geometry() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("part.opd"));
    let first = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&first) > 0);
    native_set(&mut owner, "width", "6 cm");
    let edited = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&edited) > solid_pixels(&first));
    verify_scene_undo(&mut owner, &mut window, &first, &edited);
    verify_failed_scene(&mut owner, &mut window);
    verify_closed_scene(&mut owner, &mut window, document);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_scene_undo(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    first: &ViewportCapture,
    edited: &ViewportCapture,
) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(publish_native_scene(owner, window).rgba, first.rgba);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(publish_native_scene(owner, window).rgba, edited.rgba);
}
fn verify_failed_scene(owner: &mut NativeDesktopOwner, window: &mut NativeHeadWindow) {
    native_set(owner, "depth", "0 cm");
    let failed = publish_native_scene(owner, window);
    assert_eq!(solid_pixels(&failed), 0);
    assert!(
        window
            .view
            .native_title()
            .unwrap()
            .contains("1 unavailable features")
    );
    assert!(matches!(
        window.scene_status().unwrap().health[0].status,
        oblikovati_model::feature::FeatureEvaluationStatus::Failed(_)
    ));
}

fn verify_closed_scene(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    document: u64,
) {
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    let closed = publish_native_scene(owner, window);
    assert_eq!(solid_pixels(&closed), 0);
    assert_eq!(window.view.native_title().unwrap(), "Oblikovati");
}

#[test]
fn native_body_visibility_removes_and_restores_pixels_without_model_edits() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("visibility.opd"));
    let first = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&first) > 0);
    native_body_visibility(&mut owner, false);
    assert_eq!(
        solid_pixels(&publish_native_scene(&mut owner, &mut window)),
        0
    );
    native_body_visibility(&mut owner, true);
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        first.rgba
    );
    assert!(!owner.documents().workspace().needs_save(document).unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn native_body_visibility(owner: &mut NativeDesktopOwner, visible: bool) {
    let result = Client::new(owner.document_host())
        .body()
        .set_visible(&BodySetVisibleArgs {
            body_index: 0,
            visible,
        })
        .unwrap();
    assert_eq!(result.body.visible, visible);
}

#[test]
fn native_body_deletion_and_undo_remove_and_restore_window_pixels() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("delete.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&original) > 0);
    let reply = Client::new(owner.document_host())
        .body()
        .delete(&BodyIndexArgs { body_index: 0 })
        .unwrap();
    assert!(reply.bodies.unwrap_or_default().is_empty());
    assert_eq!(
        solid_pixels(&publish_native_scene(&mut owner, &mut window)),
        0
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        original.rgba
    );
    assert!(owner.documents().workspace().needs_save(document).unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn native_typed_prism_failure_clears_pixels_and_recovers_exact_scene() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("typed-failure.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    native_set(&mut owner, "width", "0.000000000000001 cm");
    let failed = publish_native_scene(&mut owner, &mut window);
    assert_eq!(solid_pixels(&failed), 0);
    verify_typed_native_failure(window.scene_status().unwrap());
    native_set(&mut owner, "width", "4 cm");
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        original.rgba
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_typed_native_failure(status: &crate::HeadSceneStatus) {
    use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureFailure};
    let FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(cause)) =
        &status.health[0].status
    else {
        panic!(
            "collapsed edge: expected typed adapter failure, got {:?}",
            status.health
        );
    };
    assert_eq!(
        cause.phase(),
        oblikovati_martensite::EvaluationPhase::Profile
    );
    assert_eq!(
        cause.kind(),
        oblikovati_martensite::EvaluationFailureKind::InvalidGeometry
    );
    assert_eq!(
        cause.feature(),
        Some(oblikovati_model::feature::FeatureId(1))
    );
    assert_eq!(cause.sketch(), Some(fixture::SKETCH));
    assert!(status.title().contains("1 unavailable features"));
}
#[test]
fn native_rotation_recomputes_constraints_and_keeps_the_visible_prism() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("rotate.opd"));
    let before = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&before) > 0);
    rotate_native_profile(&mut owner);
    let after = publish_native_scene(&mut owner, &mut window);
    assert_eq!(after.rgba, before.rgba);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn rotate_native_profile(owner: &mut NativeDesktopOwner) {
    let result = Client::new(owner.document_host())
        .sketch()
        .transform(&TransformSketchArgs {
            op: "rotate".into(),
            entities: Some((101..105).collect()),
            center: Some(vec![1.0, 1.0]),
            angle: "15 deg".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(result.created.is_none());
}
#[path = "native_scene/copy.rs"]
mod copy_tests;
#[path = "native_scene/polyline.rs"]
mod polyline_tests;
#[path = "native_scene/split.rs"]
mod split_tests;
