// SPDX-License-Identifier: GPL-2.0-only
use super::geometric_tools::{
    geometric_owner, geometry_for_relation as relation_geometry, inspection,
};
use super::slot_input::deliver;
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_gui::GuiSketchConstraint::{self, Collinear, Concentric, Equal, Tangent};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
use oblikovati_viewport::ViewportKey;

fn circular_recipe(center: [f64; 2], radius: f64, arc: bool) -> SketchPlacementRecipe {
    if !arc {
        return SketchPlacementRecipe::circle(center, radius).unwrap();
    }
    let ends = [
        [center[0] + radius, center[1]],
        [center[0], center[1] + radius],
    ];
    SketchPlacementRecipe::arc(center, ends, true).unwrap()
}
fn circular_owner(arcs: [bool; 2]) -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    for (center, radius, arc) in [([0., 0.], 100., arcs[0]), ([240., 70.], 60., arcs[1])] {
        owner
            .commit_sketch_placement(document, sketch, &circular_recipe(center, radius, arc), &[])
            .unwrap();
    }
    (owner, document, sketch)
}
fn circular_pixels(arcs: [bool; 2]) -> [[i32; 2]; 2] {
    [
        if arcs[0] { [71, 71] } else { [100, 0] },
        if arcs[1] { [282, 112] } else { [300, 70] },
    ]
}
fn apply_curve_tool(owner: &mut TestOwner, choice: GuiSketchConstraint, pixels: [[i32; 2]; 2]) {
    owner.constrain_sketch_from_gui(choice).unwrap();
    deliver(owner, &[button(pixels[0], true), button(pixels[1], true)]);
    assert!(
        owner.sketch_tool.is_none(),
        "{choice:?}: {:?}",
        owner.command_notice
    );
}
fn assert_curve_kind(geometry: &Sketch2D, choice: GuiSketchConstraint) {
    let relation = &geometry.constraints().last().unwrap().relation;
    assert!(
        matches!(
            (choice, relation),
            (Concentric, PlanarConstraint::Concentric { .. })
                | (Equal, PlanarConstraint::EqualRadius { .. })
                | (Tangent, PlanarConstraint::CircularTangent { .. })
        ),
        "{choice:?}: {relation:?}"
    );
}
fn verify_circular_pair(arcs: [bool; 2], choice: GuiSketchConstraint) {
    let (mut owner, document, sketch) = circular_owner(arcs);
    let before = inspection(&mut owner).dof;
    apply_curve_tool(&mut owner, choice, circular_pixels(arcs));
    let geometry = relation_geometry(&owner, document, sketch);
    assert_curve_kind(geometry, choice);
    assert_circular_solution(geometry);
    let after = inspection(&mut owner).dof;
    assert_eq!(after, before - if choice == Concentric { 2 } else { 1 });
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(inspection(&mut owner).dof, before);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(inspection(&mut owner).dof, after);
}
#[test]
fn circular_relation_tools_admit_circle_arc_pairs_and_preserve_history() {
    for arcs in [[false, false], [false, true], [true, false], [true, true]] {
        for choice in [Concentric, Equal, Tangent] {
            verify_circular_pair(arcs, choice);
        }
    }
}
#[test]
fn line_relation_tools_create_collinear_or_equal_length_relations() {
    for (choice, expected) in [
        (Collinear, LineRelation::Collinear),
        (Equal, LineRelation::EqualLength),
    ] {
        let (mut owner, document, sketch) =
            geometric_owner(&[[0., 0.], [100., 20.], [140., 120.]], true);
        apply_curve_tool(&mut owner, choice, [[50, 10], [120, 70]]);
        let relation = &relation_geometry(&owner, document, sketch)
            .constraints()
            .last()
            .unwrap()
            .relation;
        assert!(
            matches!(relation, PlanarConstraint::Lines { relation, .. } if *relation == expected)
        );
        assert_eq!(inspection(&mut owner).dof, 5);
    }
}
fn line_curve_owner(arc: bool) -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, document, sketch) = geometric_owner(&[[-150., 150.], [150., 170.]], true);
    let recipe = circular_recipe([0., 0.], 100., arc);
    owner
        .commit_sketch_placement(document, sketch, &recipe, &[])
        .unwrap();
    (owner, document, sketch)
}
fn verify_line_curve_pair(arc: bool, reverse: bool) {
    let (mut owner, document, sketch) = line_curve_owner(arc);
    let before = inspection(&mut owner).dof;
    let mut pixels = [[0, 160], if arc { [71, 71] } else { [100, 0] }];
    if reverse {
        pixels.reverse();
    }
    apply_curve_tool(&mut owner, Tangent, pixels);
    let geometry = relation_geometry(&owner, document, sketch);
    let PlanarConstraint::LineTangent { line, curve, .. } =
        geometry.constraints().last().unwrap().relation
    else {
        panic!("expected line/circular tangent")
    };
    assert_line_contact(geometry, line, curve);
    assert_eq!(inspection(&mut owner).dof, before - 1);
}
#[test]
fn tangent_tool_normalizes_line_and_circular_picks_in_either_order() {
    for arc in [false, true] {
        for reverse in [false, true] {
            verify_line_curve_pair(arc, reverse);
        }
    }
}
fn circular_frame(geometry: &Sketch2D, id: SketchEntityId) -> ([f64; 2], f64) {
    if geometry.entity_kind(id).unwrap() == oblikovati_api::types::SKETCH_ENTITY_CIRCLE {
        let circle = geometry.circle(id).unwrap();
        return (
            geometry.point(circle.center).unwrap().position,
            circle.radius,
        );
    }
    let arc = geometry.arc(id).unwrap();
    (
        geometry.point(arc.center).unwrap().position,
        geometry.arc_radius(id).unwrap(),
    )
}
fn assert_circular_solution(geometry: &Sketch2D) {
    let relation = &geometry.constraints().last().unwrap().relation;
    let (PlanarConstraint::Concentric { curves }
    | PlanarConstraint::EqualRadius { curves }
    | PlanarConstraint::CircularTangent { curves, .. }) = relation
    else {
        panic!("expected circular relation: {relation:?}")
    };
    let [(first, a), (second, b)] = curves.map(|id| circular_frame(geometry, id));
    let distance = (first[0] - second[0]).hypot(first[1] - second[1]);
    let residual = match relation {
        PlanarConstraint::Concentric { .. } => distance,
        PlanarConstraint::EqualRadius { .. } => a - b,
        PlanarConstraint::CircularTangent {
            contact: CircleContact::External,
            ..
        } => distance - (a + b),
        PlanarConstraint::CircularTangent { .. } => distance - (a - b).abs(),
        _ => unreachable!(),
    };
    assert!(residual.abs() < 1e-7, "{relation:?}: residual {residual}");
}
fn assert_line_contact(geometry: &Sketch2D, line: SketchEntityId, curve: SketchEntityId) {
    let [first, second] = geometry
        .line(line)
        .unwrap()
        .endpoints
        .map(|point| geometry.point(point).unwrap().position);
    let (center, radius) = circular_frame(geometry, curve);
    let direction: [f64; 2] = std::array::from_fn(|axis| second[axis] - first[axis]);
    let cross = direction[0] * (center[1] - first[1]) - direction[1] * (center[0] - first[0]);
    let distance = cross.abs() / direction[0].hypot(direction[1]);
    assert!(
        (distance - radius).abs() < 1e-7,
        "line {line:?}, curve {curve:?}: distance {distance}, radius {radius}"
    );
}
#[test]
fn relation_tools_filter_points_and_cancel_without_history() {
    for choice in [Collinear, Equal, Concentric, Tangent] {
        let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 20.]], false);
        let history = owner.gui_command_state().unwrap().undo_label;
        owner.constrain_sketch_from_gui(choice).unwrap();
        deliver(&mut owner, &[button([0, 0], true), button([100, 20], true)]);
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            0
        );
        deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
        assert!(owner.sketch_tool.is_none());
        assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    }
}
#[test]
fn equal_mixed_first_pair_waits_for_a_matching_curve_category() {
    let (mut owner, document, sketch) = line_curve_owner(false);
    let second = SketchPlacementRecipe::line_chain(&[[300., 150.], [400., 220.]], false).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &second, &[])
        .unwrap();
    let before = inspection(&mut owner).dof;
    owner.constrain_sketch_from_gui(Equal).unwrap();
    deliver(
        &mut owner,
        &[button([100, 0], true), button([0, 160], true)],
    );
    assert!(owner.sketch_tool.is_some());
    assert_eq!(inspection(&mut owner).dof, before);
    deliver(&mut owner, &[button([350, 185], true)]);
    assert!(owner.sketch_tool.is_none());
    let relation = &relation_geometry(&owner, document, sketch)
        .constraints()
        .last()
        .unwrap()
        .relation;
    assert!(matches!(
        relation,
        PlanarConstraint::Lines {
            relation: LineRelation::EqualLength,
            ..
        }
    ));
    assert_eq!(inspection(&mut owner).dof, before - 1);
}

#[test]
fn curve_only_and_line_only_tools_ignore_the_other_category() {
    for (choice, pixel) in [(Concentric, [0, 160]), (Collinear, [100, 0])] {
        let (mut owner, _, _) = line_curve_owner(false);
        owner.constrain_sketch_from_gui(choice).unwrap();
        deliver(&mut owner, &[button(pixel, true)]);
        assert_eq!(
            Client::new(owner.document_host())
                .model()
                .selection()
                .unwrap()
                .count,
            0
        );
        assert!(owner.sketch_tool.is_some());
    }
}
fn fix_circle_centres(owner: &mut TestOwner) {
    for pixel in [[0, 0], [240, 70]] {
        owner
            .constrain_sketch_from_gui(GuiSketchConstraint::Fix)
            .unwrap();
        deliver(owner, &[button(pixel, true)]);
    }
}
#[test]
fn concentric_refusal_keeps_picks_and_the_original_recipe() {
    let (mut owner, document, _) = circular_owner([false, false]);
    fix_circle_centres(&mut owner);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let history = owner.gui_command_state().unwrap().undo_label;
    owner.constrain_sketch_from_gui(Concentric).unwrap();
    deliver(
        &mut owner,
        &[button([100, 0], true), button([300, 70], true)],
    );
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::AuthoringFailed { .. })
    ));
    assert!(owner.sketch_tool.is_some());
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        2
    );
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .same_recipe(&before)
    );
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
}
