// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input_tests::{PlacementEventWindow, verify_outline_pixels};
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use std::collections::VecDeque;
#[test]
fn native_recipe_tools_preview_commit_finish_and_undo_through_shared_input_pipeline() {
    for kind in [
        SketchPlacementTool::Circle,
        SketchPlacementTool::Ellipse,
        SketchPlacementTool::ThreePointCircle {
            options: oblikovati_model::sketch::CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
        SketchPlacementTool::CenterPointArc,
        SketchPlacementTool::ThreePointArc {
            options: oblikovati_model::sketch::CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
        SketchPlacementTool::Rectangle,
        SketchPlacementTool::CenterRectangle,
        SketchPlacementTool::ThreePointRectangle,
    ] {
        run_native_recipe_tool(kind);
    }
}
fn run_native_recipe_tool(kind: SketchPlacementTool) {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("recipe-placement.opd"));
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
    if matches!(
        kind,
        SketchPlacementTool::ThreePointRectangle
            | SketchPlacementTool::ThreePointCircle { .. }
            | SketchPlacementTool::ThreePointArc { .. }
            | SketchPlacementTool::CenterPointArc
            | SketchPlacementTool::Ellipse
    ) {
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
        } else if matches!(
            kind,
            SketchPlacementTool::Circle
                | SketchPlacementTool::ThreePointCircle { .. }
                | SketchPlacementTool::ThreePointArc { .. }
                | SketchPlacementTool::CenterPointArc
                | SketchPlacementTool::Ellipse
        ) {
            1
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
    commit_native_recipe(&mut owner, &mut window, document, &preview);
    let committed = window
        .native
        .sketch_scene_status()
        .unwrap()
        .sketches
        .clone();
    owner.finish_sketch_edit().unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(
        window.native.sketch_scene_status().unwrap().sketches,
        committed
    );
    assert!(window.native.sketch_display_declines().unwrap().is_empty());

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
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(
        window.native.sketch_scene_status().unwrap().sketches,
        committed
    );
    capture_native_scene(&mut window.native);
    super::placement_recipe_tests::verify_reopened_placement(
        &mut owner,
        document,
        &directory.path().join("recipe-placement.opd"),
    );
    publish_native_scene(&mut owner, &mut window.native);
    assert_eq!(
        window.native.sketch_scene_status().unwrap().sketches,
        committed
    );
    assert!(window.native.sketch_display_declines().unwrap().is_empty());
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn commit_native_recipe(
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
    crate::placement_test_samples::assert_recipe_tail(geometry, preview);
    capture_native_scene(&mut window.native);
}
