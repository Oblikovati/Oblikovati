// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    rectangle_input::placement_owner,
    slot_input::{PlanarInputWindow, deliver},
};
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    sketch::Sketch2D,
};
use oblikovati_viewport::{ViewportEvent, ViewportKey};
fn point_sketches(owner: &TestOwner, document: u64) -> &DocumentSketches {
    &owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
}
#[test]
fn point_hover_does_not_edit_and_first_press_commits_exactly_once() {
    let (mut owner, document, sketch) = placement_owner(SketchPlacementTool::Point);
    let before = owner.gui_command_state().unwrap();
    deliver(&mut owner, &[motion([3, -4])]);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    let preview = owner.placement_preview().unwrap().clone();
    deliver(&mut owner, &[button([3, -4], true)]);
    assert!(!owner.placement_active());
    assert!(owner.placement_preview().is_none());
    let saved = point_sketches(&owner, document).clone();
    let geometry = saved.get(sketch).unwrap().geometry();
    assert_single_point(geometry, [3., -4.]);
    crate::placement_test_samples::assert_recipe_tail(geometry, &preview);
    deliver(
        &mut owner,
        &[
            motion([30, 40]),
            button([30, 40], false),
            button([9, 9], true),
        ],
    );
    assert!(point_sketches(&owner, document).same_recipe(&saved));
    verify_point_undo(&mut owner, document, sketch, &saved);
}
fn assert_single_point(geometry: &Sketch2D, position: [f64; 2]) {
    assert_eq!(geometry.points().len(), 1);
    let point = geometry.points()[0];
    assert_eq!(point.position, position);
    assert_eq!(geometry.entity_ids(), &[point.id]);
    assert_eq!(geometry.owned_entity_ids(), vec![point.id]);
}
fn verify_point_undo(
    owner: &mut TestOwner,
    document: u64,
    sketch: DocumentSketchId,
    saved: &DocumentSketches,
) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        point_sketches(owner, document)
            .get(sketch)
            .unwrap()
            .geometry()
            .points()
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert!(point_sketches(owner, document).same_recipe(saved));
}
#[test]
fn point_can_commit_without_prior_motion_and_requires_activation_for_each_point() {
    let (mut owner, document, sketch) = placement_owner(SketchPlacementTool::Point);
    deliver(&mut owner, &[button([3, 4], true), button([3, 4], false)]);
    assert_single_point(
        point_sketches(&owner, document)
            .get(sketch)
            .unwrap()
            .geometry(),
        [3., 4.],
    );
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementPrompt { message }) if message == "Point placed.")
    );
    owner
        .start_sketch_placement(document, sketch, SketchPlacementTool::Point)
        .unwrap();
    deliver(&mut owner, &[button([3, 4], true), button([3, 4], false)]);
    let geometry = point_sketches(&owner, document)
        .get(sketch)
        .unwrap()
        .geometry();
    assert_eq!(geometry.points().len(), 2);
    assert_ne!(geometry.points()[0].id, geometry.points()[1].id);
    assert_eq!(geometry.points()[0].position, geometry.points()[1].position);
}
#[test]
fn cancel_and_lost_focus_drop_point_preview_without_changing_history() {
    for event in [
        key(ViewportKey::Cancel, true),
        ViewportEvent::Focus(false),
        ViewportEvent::Mapped(false),
        ViewportEvent::CloseRequested,
    ] {
        let (mut owner, document, _) = placement_owner(SketchPlacementTool::Point);
        let before = point_sketches(&owner, document).clone();
        deliver(&mut owner, &[motion([3, 4]), event]);
        assert!(!owner.placement_active());
        assert!(owner.placement_preview().is_none());
        assert!(point_sketches(&owner, document).same_recipe(&before));
    }
}
#[test]
fn projection_failure_or_edge_on_plane_preserves_point_tool_for_retry() {
    for window in [
        PlanarInputWindow {
            fail: true,
            edge_on: false,
        },
        PlanarInputWindow {
            fail: false,
            edge_on: true,
        },
    ] {
        let (mut owner, document, sketch) = placement_owner(SketchPlacementTool::Point);
        let before = owner.gui_command_state().unwrap();
        owner
            .process_window_inputs(&[button([3, 4], true)], &window)
            .unwrap();
        assert!(owner.placement_active());
        assert!(owner.placement_preview().is_none());
        assert_eq!(owner.gui_command_state().unwrap(), before);
        deliver(&mut owner, &[button([3, 4], true)]);
        assert!(!owner.placement_active());
        assert_single_point(
            point_sketches(&owner, document)
                .get(sketch)
                .unwrap()
                .geometry(),
            [3., 4.],
        );
    }
}
