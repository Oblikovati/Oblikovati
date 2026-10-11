// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchError, SketchPoint2D, SketchRelationId, SketchSolveOutcome,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "3 cm")
        .unwrap();
    parameters
}
fn point(id: u64, position: [f64; 2]) -> SketchPoint2D {
    SketchPoint2D {
        id: SketchEntityId(id),
        position,
    }
}
fn geometry(points: &[(u64, [f64; 2])]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in points {
        sketch.add_point(SketchEntityId(*id), *position).unwrap();
    }
    sketch
}
fn relation(sketch: &mut Sketch2D, id: u64, relation: PlanarConstraint) {
    sketch
        .add_constraint(SketchConstraintId(id), relation)
        .unwrap();
}
fn driven_segment() -> Sketch2D {
    let mut sketch = geometry(&[(1, [0.0; 2]), (2, [2.0, 1.0])]);
    let points = [SketchEntityId(1), SketchEntityId(2)];
    relation(&mut sketch, 1, PlanarConstraint::Horizontal { points });
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Distance {
            points,
            parameter: 1,
            measure: DistanceMeasure::Horizontal,
        },
    );
    sketch
}
fn solve(sketch: &Sketch2D, anchors: &[SketchPoint2D]) -> SketchSolveOutcome {
    sketch
        .solve_with_grounded_points(&parameters(), anchors, SolveOptions::default(), || false)
        .unwrap()
}
fn assert_exact_anchor(result: &SketchSolveOutcome, anchor: SketchPoint2D) {
    let position = result.candidate().point(anchor.id).unwrap().position;
    assert_eq!(
        position.map(f64::to_bits),
        anchor.position.map(f64::to_bits)
    );
}
fn assert_solved(result: &SketchSolveOutcome, active: usize) {
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().evaluation.variable_count(), active);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
}

#[test]
fn grounded_reference_is_exact_and_has_no_numerical_variables() {
    let sketch = geometry(&[(1, [7.0, 8.0])]);
    let anchor = point(1, [-0.0, 3.125]);
    let result = solve(&sketch, &[anchor]);
    assert_solved(&result, 0);
    assert_exact_anchor(&result, anchor);
    assert_eq!(sketch.point(anchor.id).unwrap().position, [7.0, 8.0]);
    assert_eq!(result.candidate().constraints().len(), 0);
}
#[test]
fn current_grounded_coordinates_drive_authored_constraints_without_new_fix_relations() {
    let sketch = driven_segment();
    let anchor = point(1, [10.0, -4.0]);
    let result = solve(&sketch, &[anchor]);
    assert_solved(&result, 2);
    assert_exact_anchor(&result, anchor);
    assert_eq!(result.candidate().constraints(), sketch.constraints());
    assert_eq!(result.parameter_dependencies(), [1]);
    let endpoint = result
        .candidate()
        .point(SketchEntityId(2))
        .unwrap()
        .position;
    assert!((endpoint[0] - 7.0).abs() < 1e-7);
    assert!((endpoint[1] + 4.0).abs() < 1e-7);
}
#[test]
fn analysis_uses_current_anchors_without_changing_the_input() {
    let sketch = driven_segment();
    let before = sketch.recipe();
    let structure = sketch
        .analyze_with_grounded_points(&parameters(), &[point(1, [10.0, -4.0])], 1e-9)
        .unwrap();
    assert_eq!(structure.rank, 2);
    assert_eq!(structure.degrees_of_freedom, 0);
    assert_eq!(before, sketch.recipe());
}
#[test]
fn unrelated_free_points_remain_in_the_variable_universe() {
    let sketch = geometry(&[(1, [0.0; 2]), (2, [6.0, 7.0])]);
    let result = solve(&sketch, &[point(1, [2.0, 3.0])]);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 2);
    assert_eq!(result.numerical().evaluation.variable_count(), 2);
    assert_eq!(
        result
            .candidate()
            .point(SketchEntityId(2))
            .unwrap()
            .position,
        [6.0, 7.0]
    );
}
#[test]
fn fully_grounded_conflict_keeps_authored_source_identity_and_both_anchors() {
    let mut sketch = geometry(&[(1, [0.0; 2]), (2, [1.0, 0.0])]);
    relation(
        &mut sketch,
        9,
        PlanarConstraint::Coincident {
            points: [SketchEntityId(1), SketchEntityId(2)],
        },
    );
    let anchors = [point(1, [2.0, 3.0]), point(2, [5.0, 6.0])];
    let result = solve(&sketch, &anchors);
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(
        result.conflicting_relations(),
        [SketchRelationId::Authored(SketchConstraintId(9))]
    );
    for anchor in anchors {
        assert_exact_anchor(&result, anchor);
    }
}
#[path = "sketch_grounded_points/circular.rs"]
mod circular;
#[path = "sketch_grounded_points/guards.rs"]
mod guards;
