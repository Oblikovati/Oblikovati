// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{rectangle_input::placement_owner, slot_input::deliver};
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    sketch::{CircumcircleOptions, Sketch2D, SketchPlacementGeometry, SketchPlacementRecipe},
};
use oblikovati_viewport::ViewportKey;
fn three_point_circle(bound: f64) -> SketchPlacementTool {
    SketchPlacementTool::ThreePointCircle {
        options: CircumcircleOptions {
            maximum_point_residual: bound,
        },
    }
}
fn owned_sketches(owner: &TestOwner, document: u64) -> &DocumentSketches {
    &owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
}
#[test]
fn circles_share_click_drag_preview_commit_restart_and_history() {
    for kind in [SketchPlacementTool::Circle, three_point_circle(0.)] {
        let clicked = place_round_tool(kind, false, verify_circle_preview);
        let dragged = place_round_tool(kind, true, verify_circle_preview);
        assert!(clicked.same_recipe(&dragged));
    }
}
pub(super) fn place_round_tool(
    kind: SketchPlacementTool,
    drag: bool,
    verify: fn(&Sketch2D, &SketchPlacementRecipe),
) -> DocumentSketches {
    let (mut owner, document, sketch) = placement_owner(kind);
    let before = owner.gui_command_state().unwrap();
    round_tool_prefix(&mut owner, kind, drag);
    deliver(&mut owner, &[motion([5, 5])]);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    let preview = owner.placement_preview().unwrap().clone();
    deliver(
        &mut owner,
        &[button([5, 5], !drag || kind.point_count() != 2)],
    );
    assert!(owner.placement_active());
    assert!(owner.placement_preview().is_none());
    let saved = owned_sketches(&owner, document).clone();
    verify(saved.get(sketch).unwrap().geometry(), &preview);
    circle_history(&mut owner, document, sketch, &saved);
    saved
}
fn verify_circle_preview(geometry: &Sketch2D, preview: &SketchPlacementRecipe) {
    assert_eq!(geometry.circles().len(), 1);
    let SketchPlacementGeometry::Circle { radius, .. } = preview.members[0].geometry else {
        panic!("expected circle")
    };
    assert_eq!(geometry.circles()[0].radius, radius);
    assert_eq!(geometry.points()[0].position, preview.points[0]);
}
fn round_tool_prefix(owner: &mut TestOwner, kind: SketchPlacementTool, drag: bool) {
    deliver(owner, &[button([0, 0], true)]);
    if !drag {
        deliver(owner, &[button([0, 0], false)]);
    }
    if kind.point_count() != 2 {
        deliver(
            owner,
            &[
                motion([10, 0]),
                button([10, 0], !drag),
                button([10, 0], false),
            ],
        );
    }
}
fn circle_history(
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
        owned_sketches(owner, document)
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
    assert!(owned_sketches(owner, document).same_recipe(saved));
}
#[test]
fn bad_accuracy_does_not_activate_a_circle_tool() {
    for bound in [-1., f64::NAN, f64::INFINITY] {
        let (mut owner, document, sketch) = placement_owner(SketchPlacementTool::Circle);
        assert!(
            owner
                .start_sketch_placement(document, sketch, three_point_circle(bound))
                .is_err()
        );
        assert!(owner.placement_active());
    }
}
#[test]
fn repeated_and_collinear_points_preserve_stage_for_retry_and_escape() {
    let (mut owner, document, _) = placement_owner(three_point_circle(0.));
    let before = owned_sketches(&owner, document).clone();
    click(&mut owner, [0, 0]);
    click(&mut owner, [0, 0]);
    assert_placement_failed(&mut owner);
    click(&mut owner, [10, 0]);
    click(&mut owner, [5, 0]);
    assert_placement_failed(&mut owner);
    assert!(owned_sketches(&owner, document).same_recipe(&before));
    deliver(&mut owner, &[motion([5, 5])]);
    assert!(owner.placement_preview().is_some());
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
    assert!(!owner.placement_active());
    assert!(owned_sketches(&owner, document).same_recipe(&before));
}
fn assert_placement_failed(owner: &mut TestOwner) {
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::PlacementFailed { .. })
    ));
    assert!(owner.placement_preview().is_none());
}
fn click(owner: &mut TestOwner, pixel: [i32; 2]) {
    deliver(owner, &[button(pixel, true), button(pixel, false)]);
}
