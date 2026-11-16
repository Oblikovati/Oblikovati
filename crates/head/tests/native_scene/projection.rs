// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{SketchPlacementRecipe, SketchPlane};
#[test]
fn native_cursor_positions_feed_the_owned_slot_commit_and_undo_path() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("cursor.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    commit_pointer_slot(&mut owner, &window, document);
    verify_intersection_and_undo(&mut owner, &mut window, &original);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn commit_pointer_slot(owner: &mut NativeDesktopOwner, window: &NativeHeadWindow, document: u64) {
    let plane = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .resolve_sketch_plane(fixture::SKETCH)
        .unwrap();
    let centers =
        [[650, 430], [820, 500]].map(|pixel| window.sketch_point(pixel, plane).unwrap().unwrap());
    verify_pointer_roundtrip(window, plane, centers);
    let recipe = SketchPlacementRecipe::straight_slot(centers, 0.5).unwrap();
    let report = owner
        .commit_sketch_placement(document, fixture::SKETCH, &recipe, &[])
        .unwrap();
    verify_slot_geometry(owner, document, &report.points, centers);
}
fn verify_slot_geometry(
    owner: &NativeDesktopOwner,
    document: u64,
    points: &[oblikovati_model::sketch::SketchEntityId],
    centers: [[f64; 2]; 2],
) {
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let geometry = snapshot.sketches.get(fixture::SKETCH).unwrap().geometry();
    verify_slot_crosses_rectangle(geometry, points[0], points[1]);
    for (index, expected) in [(5, centers[0]), (4, centers[1])] {
        let actual = geometry.point(points[index]).unwrap().position;
        for (a, b) in actual.into_iter().zip(expected) {
            assert!((a - b).abs() < 1e-8);
        }
    }
}
fn verify_pointer_roundtrip(window: &NativeHeadWindow, plane: SketchPlane, centers: [[f64; 2]; 2]) {
    for (point, pixel) in centers.into_iter().zip([[650, 430], [820, 500]]) {
        let restored = window.sketch_pixel(plane, point).unwrap();
        for (a, b) in restored.into_iter().zip(pixel) {
            assert!((a - b).abs() <= 1);
        }
    }
}

fn verify_slot_crosses_rectangle(
    geometry: &oblikovati_model::sketch::Sketch2D,
    first: oblikovati_model::sketch::SketchEntityId,
    last: oblikovati_model::sketch::SketchEntityId,
) {
    let a = geometry.point(first).unwrap().position;
    let b = geometry.point(last).unwrap().position;
    assert!(
        a[1] * b[1] < 0.0,
        "slot edge {a:?} to {b:?}: expected crossing y=0"
    );
    let x = a[0] + (b[0] - a[0]) * (-a[1] / (b[1] - a[1]));
    assert!(
        x > 0.0 && x < 4.0,
        "slot crossing x={x}: expected the rectangle edge interior"
    );
}
fn verify_intersection_and_undo(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    original: &ViewportCapture,
) {
    let after = publish_native_scene(owner, window);
    assert_eq!(solid_pixels(&after), 0);
    verify_intersection_health(window.scene_status().unwrap());
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(publish_native_scene(owner, window).rgba, original.rgba);
}
fn verify_intersection_health(status: &crate::HeadSceneStatus) {
    use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureFailure};
    let FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(cause)) =
        &status.health[0].status
    else {
        panic!(
            "intersecting slot: expected kernel profile decline, got {:?}",
            status.health
        );
    };
    verify_profile_decline(cause);
}
fn verify_profile_decline(cause: &oblikovati_martensite::EvaluationFailure) {
    assert_eq!(
        cause.phase(),
        oblikovati_martensite::EvaluationPhase::Profile
    );
    assert_eq!(
        cause.kind(),
        oblikovati_martensite::EvaluationFailureKind::InvalidGeometry
    );
    assert_eq!(cause.sketch(), Some(fixture::SKETCH));
    let source = std::error::Error::source(cause).unwrap().to_string();
    assert!(source.contains("SelfIntersecting"), "{source}");
    assert!(source.contains("ProfileCurveId(101)"), "{source}");
    assert!(source.contains("ProfileCurveId(111)"), "{source}");
}
