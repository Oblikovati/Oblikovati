// SPDX-License-Identifier: GPL-2.0-only
use super::geometric_tools::{geometric_owner, inspection};
use super::slot_input::deliver;
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_gui::GuiSketchConstraint::{self, Fix, Horizontal, Parallel, Perpendicular};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
use oblikovati_viewport::ViewportKey;

fn relation_owner() -> (TestOwner, u64, DocumentSketchId) {
    geometric_owner(&[[0., 0.], [100., 20.], [140., 120.]], true)
}
fn selected_count(owner: &mut TestOwner) -> usize {
    let selection = Client::new(owner.document_host())
        .model()
        .selection()
        .unwrap();
    usize::try_from(selection.count).unwrap()
}
fn start_first_pick(owner: &mut TestOwner, choice: GuiSketchConstraint) {
    owner.constrain_sketch_from_gui(choice).unwrap();
    deliver(owner, &[button([50, 10], true)]);
    assert_eq!(selected_count(owner), 1);
}
fn finish_second_pick(owner: &mut TestOwner) {
    deliver(owner, &[button([120, 70], true)]);
}
fn assert_relation_history(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(inspection(owner).dof, 6);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(inspection(owner).dof, 5);
}
#[test]
fn two_line_relations_require_distinct_lines_and_commit_one_history_step() {
    for choice in [Parallel, Perpendicular] {
        let (mut owner, document, sketch) = relation_owner();
        start_first_pick(&mut owner, choice);
        deliver(&mut owner, &[button([50, 10], true)]);
        assert_eq!(inspection(&mut owner).dof, 6);
        assert_eq!(selected_count(&mut owner), 1);
        finish_second_pick(&mut owner);
        assert_eq!(inspection(&mut owner).dof, 5);
        assert_line_relation(&owner, document, sketch, choice);
        assert!(owner.sketch_tool.is_none(), "{:?}", owner.command_notice);
        assert_relation_history(&mut owner);
        assert_line_relation(&owner, document, sketch, choice);
    }
}
#[test]
fn two_line_relations_cancel_and_replace_without_creating_constraints() {
    for choice in [Parallel, Perpendicular] {
        let (mut owner, _, _) = relation_owner();
        let history = owner.gui_command_state().unwrap().undo_label;
        start_first_pick(&mut owner, choice);
        deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
        assert!(owner.sketch_tool.is_none());
        start_first_pick(&mut owner, choice);
        owner.constrain_sketch_from_gui(Horizontal).unwrap();
        assert_eq!(selected_count(&mut owner), 0);
        assert_eq!(inspection(&mut owner).dof, 6);
        assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    }
}
#[test]
fn two_line_relations_ignore_isolated_points() {
    for choice in [Parallel, Perpendicular] {
        let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 20.]], false);
        owner.constrain_sketch_from_gui(choice).unwrap();
        deliver(&mut owner, &[button([0, 0], true), button([100, 20], true)]);
        assert_eq!(selected_count(&mut owner), 0);
        assert_eq!(inspection(&mut owner).dof, 4);
    }
}
fn relation_geometry(owner: &TestOwner, document: u64, sketch: DocumentSketchId) -> &Sketch2D {
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
fn unit_line_direction(geometry: &Sketch2D, id: SketchEntityId) -> [f64; 2] {
    let endpoints = geometry
        .line(id)
        .unwrap()
        .endpoints
        .map(|point| geometry.point(point).unwrap().position);
    let difference: [f64; 2] = std::array::from_fn(|axis| endpoints[1][axis] - endpoints[0][axis]);
    let length = difference[0].hypot(difference[1]);
    difference.map(|value| value / length)
}
fn assert_solved_directions(geometry: &Sketch2D, lines: [SketchEntityId; 2], parallel: bool) {
    let [first, second] = lines.map(|id| unit_line_direction(geometry, id));
    let residual = if parallel {
        first[0] * second[1] - first[1] * second[0]
    } else {
        first[0] * second[0] + first[1] * second[1]
    };
    assert!(
        residual.abs() < 1e-9,
        "parallel={parallel}: normalized residual {residual}"
    );
}
fn assert_line_relation(
    owner: &TestOwner,
    document: u64,
    sketch: DocumentSketchId,
    choice: GuiSketchConstraint,
) {
    let geometry = relation_geometry(owner, document, sketch);
    let PlanarConstraint::Lines { lines, relation } =
        geometry.constraints().last().unwrap().relation
    else {
        panic!("expected line relation for {choice:?}")
    };
    let expected = if choice == Parallel {
        LineRelation::Parallel
    } else {
        LineRelation::Perpendicular
    };
    assert_eq!(relation, expected);
    assert_solved_directions(geometry, lines, choice == Parallel);
}
fn fix_relation_points(owner: &mut TestOwner) {
    for pixel in [[0, 0], [100, 20], [140, 120]] {
        owner.constrain_sketch_from_gui(Fix).unwrap();
        deliver(owner, &[button(pixel, true)]);
    }
    assert_eq!(inspection(owner).dof, 0);
}
#[test]
fn two_line_relations_refuse_conflicting_fixed_geometry_without_adoption() {
    for choice in [Parallel, Perpendicular] {
        let (mut owner, _, _) = relation_owner();
        fix_relation_points(&mut owner);
        let before = inspection(&mut owner);
        let history = owner.gui_command_state().unwrap().undo_label;
        start_first_pick(&mut owner, choice);
        finish_second_pick(&mut owner);
        assert!(matches!(
            owner.take_command_notice(),
            Some(DesktopCommandNotice::AuthoringFailed { .. })
        ));
        assert!(owner.sketch_tool.is_some());
        assert_eq!(inspection(&mut owner), before);
        assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    }
}
