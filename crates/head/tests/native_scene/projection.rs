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
    super::copy_tests::verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 5);
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
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry();
    for (index, expected) in [(5, centers[0]), (4, centers[1])] {
        let actual = geometry.point(report.points[index]).unwrap().position;
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
