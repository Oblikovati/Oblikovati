// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_gui::GuiSketchConstraint::{Coincident, Horizontal, Vertical};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{PlanarConstraint, Sketch2D, SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::ViewportKey;

pub(super) fn geometric_owner(
    points: &[[f64; 2]],
    connected: bool,
) -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    if connected {
        let recipe = SketchPlacementRecipe::line_chain(points, false).unwrap();
        owner
            .commit_sketch_placement(document, sketch, &recipe, &[])
            .unwrap();
    } else {
        for point in points {
            let recipe = SketchPlacementRecipe::point(*point).unwrap();
            owner
                .commit_sketch_placement(document, sketch, &recipe, &[])
                .unwrap();
        }
    }
    (owner, document, sketch)
}
pub(super) fn inspection(owner: &mut TestOwner) -> SketchInfo {
    Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs::default())
        .unwrap()
}
fn assert_tool_active(owner: &TestOwner, active: bool) {
    assert_eq!(
        matches!(
            owner.sketch_tool,
            Some(crate::sketch_tool::SketchToolState::Geometric(_))
        ),
        active
    );
}
#[test]
fn horizontal_and_vertical_line_tools_commit_through_the_typed_host_and_undo() {
    for (choice, points, pixel) in [
        (Horizontal, [[0., 0.], [100., 10.]], [50, 5]),
        (Vertical, [[0., 0.], [10., 100.]], [5, 50]),
    ] {
        let (mut owner, _, _) = geometric_owner(&points, true);
        let before = inspection(&mut owner).dof;
        owner.constrain_sketch_from_gui(choice).unwrap();
        assert_tool_active(&owner, true);
        deliver(&mut owner, &[button(pixel, true)]);
        assert_tool_active(&owner, false);
        assert_eq!(inspection(&mut owner).dof, before - 1);
        Client::new(owner.document_host())
            .transactions()
            .undo()
            .unwrap();
        assert_eq!(inspection(&mut owner).dof, before);
        Client::new(owner.document_host())
            .transactions()
            .redo()
            .unwrap();
        assert_eq!(inspection(&mut owner).dof, before - 1);
    }
}
#[test]
fn two_point_constraints_keep_picked_highlights_until_commit() {
    for choice in [Horizontal, Vertical, Coincident] {
        let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 50.]], false);
        let before = inspection(&mut owner).dof;
        owner.constrain_sketch_from_gui(choice).unwrap();
        deliver(&mut owner, &[button([0, 0], true)]);
        assert_tool_active(&owner, true);
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            1
        );
        assert_eq!(
            owner
                .gui_command_state()
                .unwrap()
                .sketch_selection
                .entities
                .len(),
            1
        );
        deliver(&mut owner, &[button([0, 0], true)]);
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            1
        );
        deliver(&mut owner, &[button([100, 50], true)]);
        assert_tool_active(&owner, false);
        assert_eq!(
            inspection(&mut owner).dof,
            before - if choice == Coincident { 2 } else { 1 }
        );
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            0
        );
    }
}
#[test]
fn cancel_and_tool_replacement_retire_picks_and_preserve_model_history() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 50.]], false);
    let history = owner.gui_command_state().unwrap().undo_label;
    owner.constrain_sketch_from_gui(Coincident).unwrap();
    deliver(
        &mut owner,
        &[button([0, 0], true), key(ViewportKey::Cancel, true)],
    );
    assert_tool_active(&owner, false);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
    owner.constrain_sketch_from_gui(Horizontal).unwrap();
    deliver(&mut owner, &[button([0, 0], true)]);
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
        .unwrap();
    assert!(owner.placement_active());
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    owner.constrain_sketch_from_gui(Vertical).unwrap();
    assert!(!owner.placement_active());
    owner.finish_sketch_edit().unwrap();
    assert_tool_active(&owner, false);
}
#[test]
fn first_midpoint_snap_is_retained_and_duplicate_line_clicks_do_not_replace_it() {
    for (first, duplicate, midpoint) in [([25, 0], [50, 0], false), ([50, 0], [25, 0], true)] {
        let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 0.]], true);
        let point = SketchPlacementRecipe::point([50., 50.]).unwrap();
        owner
            .commit_sketch_placement(document, sketch, &point, &[])
            .unwrap();
        owner.constrain_sketch_from_gui(Coincident).unwrap();
        deliver(
            &mut owner,
            &[
                button(first, true),
                button(duplicate, true),
                button([50, 50], true),
            ],
        );
        assert_tool_active(&owner, false);
        let geometry = owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry();
        let last = geometry.constraints().last().unwrap();
        assert_eq!(
            matches!(last.relation, PlanarConstraint::Midpoint { .. }),
            midpoint
        );
        assert_eq!(
            matches!(last.relation, PlanarConstraint::PointOnLine { .. }),
            !midpoint
        );
    }
}

#[test]
fn coincident_accepts_circle_and_arc_outlines_through_the_same_host_operation() {
    for (recipe, pixel) in [
        (
            SketchPlacementRecipe::circle([0.; 2], 100.).unwrap(),
            [100, 0],
        ),
        (
            SketchPlacementRecipe::arc([0.; 2], [[100., 0.], [0., 100.]], true).unwrap(),
            [70, 70],
        ),
    ] {
        let (mut owner, document, sketch) = geometric_owner(&[[150., 50.]], false);
        owner
            .commit_sketch_placement(document, sketch, &recipe, &[])
            .unwrap();
        let before = inspection(&mut owner).dof;
        owner.constrain_sketch_from_gui(Coincident).unwrap();
        deliver(&mut owner, &[button(pixel, true)]);
        assert_tool_active(&owner, true);
        deliver(&mut owner, &[button([150, 50], true)]);
        assert_tool_active(&owner, false);
        assert_eq!(inspection(&mut owner).dof, before - 1);
    }
}
#[test]
fn wrong_curve_kinds_are_not_picked_by_alignment_tools() {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    let circle = SketchPlacementRecipe::circle([0.; 2], 100.).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &circle, &[])
        .unwrap();
    let history = owner.gui_command_state().unwrap().undo_label;
    for choice in [Horizontal, Vertical] {
        owner.constrain_sketch_from_gui(choice).unwrap();
        deliver(&mut owner, &[button([100, 0], true)]);
        assert_tool_active(&owner, true);
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            0
        );
        assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    }
}
#[test]
fn conflicting_constraints_keep_the_tool_and_do_not_adopt_a_sick_sketch() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 10.]], true);
    let points = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .points()
        .iter()
        .map(|point| point.id.0)
        .collect::<Vec<_>>();
    for point in points {
        Client::new(owner.document_host())
            .constrain()
            .add(&AddConstraintArgs {
                kind: oblikovati_api::types::GEO_CONSTRAINT_FIX.0.to_string(),
                entities: Some(vec![point]),
                ..Default::default()
            })
            .unwrap();
    }
    assert_eq!(inspection(&mut owner).dof, 0);
    let fixed = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry();
    assert_eq!(
        fixed
            .points()
            .iter()
            .map(|point| point.position)
            .collect::<Vec<_>>(),
        [[0., 0.], [100., 10.]]
    );
    let history = owner.gui_command_state().unwrap().undo_label;
    owner.constrain_sketch_from_gui(Horizontal).unwrap();
    deliver(&mut owner, &[button([50, 5], true)]);
    let attempted = inspection(&mut owner);
    assert!(
        matches!(
            owner.sketch_tool,
            Some(crate::sketch_tool::SketchToolState::Geometric(_))
        ),
        "pending tool: {:?}; sketch: {:?}",
        owner.command_notice,
        attempted
    );
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::AuthoringFailed { cause }) if !cause.is_empty())
    );
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    let observed = inspection(&mut owner);
    assert!(observed.healthy);
    assert_eq!(observed.dof, 0);
}
#[test]
fn history_change_cancels_a_pending_tool_and_clears_its_stale_pick_storage() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 50.]], false);
    owner.constrain_sketch_from_gui(Coincident).unwrap();
    deliver(&mut owner, &[button([0, 0], true)]);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    deliver(&mut owner, &[]);
    assert_tool_active(&owner, false);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
    let mut click = button([0, 0], true);
    if let oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
        modifiers,
        ..
    }) = &mut click
    {
        modifiers.control = true;
    }
    deliver(&mut owner, &[click]);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        1
    );
}

pub(super) fn geometry_for_relation(
    owner: &TestOwner,
    document: u64,
    sketch: DocumentSketchId,
) -> &Sketch2D {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
}
