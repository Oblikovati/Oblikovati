// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn invalid_anchor_identity_coordinate_and_duplicates_are_declined_before_mutation() {
    let sketch = geometry(&[(1, [0.0; 2]), (2, [1.0, 0.0])]);
    let before = sketch.recipe();
    for anchors in [
        vec![point(0, [0.0; 2])],
        vec![point(99, [0.0; 2])],
        vec![point(1, [f64::NAN; 2])],
        vec![point(1, [f64::INFINITY; 2])],
        vec![point(1, [0.0; 2]); 2],
    ] {
        assert_invalid_anchors(&sketch, &anchors);
        assert_eq!(before, sketch.recipe());
    }
}
fn assert_invalid_anchors(sketch: &Sketch2D, anchors: &[SketchPoint2D]) {
    let result =
        sketch
            .solve_with_grounded_points(&parameters(), anchors, SolveOptions::default(), || false);
    assert!(result.is_err());
    assert!(
        sketch
            .analyze_with_grounded_points(&parameters(), anchors, 1e-9)
            .is_err()
    );
}
#[test]
fn cancellation_retains_exact_current_anchors_and_unchanged_input() {
    let sketch = driven_segment();
    let before = sketch.recipe();
    let anchor = point(1, [9.0, -0.0]);
    let result = sketch
        .solve_with_grounded_points(&parameters(), &[anchor], SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Cancelled);
    assert_exact_anchor(&result, anchor);
    assert_eq!(before, sketch.recipe());
}
#[test]
fn current_parameter_targets_are_reread_after_grounding() {
    let sketch = driven_segment();
    let mut parameters = parameters();
    parameters.set_expression(1, "5 cm").unwrap();
    let anchors = [point(1, [0.0; 2])];
    let result = sketch
        .solve_with_grounded_points(&parameters, &anchors, SolveOptions::default(), || false)
        .unwrap();
    assert_solved(&result, 2);
    let endpoint = result
        .candidate()
        .point(SketchEntityId(2))
        .unwrap()
        .position;
    assert!((endpoint[0] - 5.0).abs() < 1e-7);
}
#[test]
fn failed_parameter_targets_retain_the_authored_constraint_decline() {
    let sketch = driven_segment();
    let mut parameters = parameters();
    let anchors = [point(1, [0.0; 2])];
    parameters.set_expression(1, "missing").unwrap();
    assert!(matches!(
        sketch.solve_with_grounded_points(&parameters, &anchors, SolveOptions::default(), || false),
        Err(SketchError::Parameter { parameter: 1, .. })
    ));
}
#[test]
fn a_line_id_cannot_be_used_as_a_grounded_point() {
    let mut sketch = driven_segment();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    assert_invalid_anchors(&sketch, &[point(3, [0.0; 2])]);
}
#[test]
fn grounded_residual_overflow_names_the_authored_constraint() {
    let mut sketch = driven_segment();
    relation(
        &mut sketch,
        7,
        PlanarConstraint::Distance {
            points: [SketchEntityId(1), SketchEntityId(2)],
            parameter: 1,
            measure: DistanceMeasure::Aligned,
        },
    );
    let anchors = [point(1, [f64::MAX; 2]), point(2, [-f64::MAX; 2])];
    let error = sketch
        .solve_with_grounded_points(&parameters(), &anchors, SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(error, SketchError::Residual { .. }));
    assert!(error.to_string().contains("expected"));
}
#[test]
fn grounding_preserves_bits_at_finite_coordinate_extremes() {
    let sketch = geometry(&[(1, [1.0, 2.0])]);
    for position in [
        [-0.0, 0.0],
        [f64::MAX, -f64::MAX],
        [f64::from_bits(1), -f64::from_bits(1)],
    ] {
        let anchor = point(1, position);
        let result = solve(&sketch, &[anchor]);
        assert_solved(&result, 0);
        assert_exact_anchor(&result, anchor);
    }
}
