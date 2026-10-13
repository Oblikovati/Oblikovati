// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input_tests::{PlacementEventWindow, verify_outline_pixels};
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use std::collections::VecDeque;
#[test]
fn native_rectangles_preview_commit_and_undo_through_shared_input_pipeline() {
    for kind in [
        SketchPlacementTool::Rectangle,
        SketchPlacementTool::CenterRectangle,
        SketchPlacementTool::ThreePointRectangle,
    ] {
        run_native_rectangle(kind);
    }
}
fn run_native_rectangle(kind: SketchPlacementTool) {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("rectangle.opd"));
    publish_native_scene(&mut owner, &mut native);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    owner
        .start_sketch_placement(document, fixture::SKETCH, kind)
        .unwrap();
    let mut window = PlacementEventWindow {
        native,
        scripted: VecDeque::new(),
    };
    let mut inputs = vec![button([650, 430], true), button([650, 430], false)];
    if kind == SketchPlacementTool::ThreePointRectangle {
        inputs.extend([button([800, 430], true), button([800, 430], false)]);
    }
    inputs.push(motion([820, 520]));
    window.scripted = VecDeque::from([inputs]);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let preview = owner.placement_preview().unwrap().clone();
    assert_eq!(
        preview.members.len(),
        if kind == SketchPlacementTool::CenterRectangle {
            6
        } else {
            4
        }
    );
    verify_outline_pixels(&owner, &mut window);
    assert!(
        before.same_recipe(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    commit_native_rectangle(&mut owner, &mut window, document, &preview);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert!(
        before.same_recipe(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    capture_native_scene(&mut window.native);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn commit_native_rectangle(
    owner: &mut NativeDesktopOwner,
    window: &mut PlacementEventWindow,
    document: u64,
    preview: &oblikovati_model::sketch::SketchPlacementRecipe,
) {
    window.scripted = VecDeque::from([vec![button([820, 520], true), button([820, 520], false)]]);
    crate::frames::run_frames(window, HeadOptions { frames: Some(1) }, owner).unwrap();
    assert!(owner.placement_active());
    assert!(owner.placement_preview().is_none());
    assert!(!window.native.set_sketch_preview(None).unwrap());
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
    let points = &geometry.points()[geometry.points().len() - preview.points.len()..];
    for (point, expected) in points.iter().zip(&preview.points) {
        for (actual, wanted) in point.position.iter().zip(expected) {
            assert!((actual - wanted).abs() < 1e-8);
        }
    }
    capture_native_scene(&mut window.native);
}
