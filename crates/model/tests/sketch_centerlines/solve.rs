// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn grounded_geometry() -> Sketch2D {
    let mut sketch = centerline_fixture();
    for (id, point) in sketch.points().to_vec().into_iter().enumerate() {
        sketch
            .add_constraint(
                SketchConstraintId(id as u64 + 1),
                PlanarConstraint::Fix {
                    point: point.id,
                    position: point.position,
                },
            )
            .unwrap();
    }
    sketch
}
#[test]
fn centerline_flags_do_not_change_numerical_rank_or_geometric_coordinates() {
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let mut sketch = grounded_geometry();
    let baseline = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    sketch.set_line_centerline(SketchEntityId(9), true).unwrap();
    let solved = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_same_numeric_evidence(&solved, &baseline);
    assert_eq!(solved.candidate().points(), baseline.candidate().points());
    assert!(solved.candidate().is_centerline(SketchEntityId(9)).unwrap());
}
#[test]
fn finite_axis_detection_is_independent_of_coordinate_scale() {
    for scale in [f64::MIN_POSITIVE, 1.0, 1e150, f64::MAX] {
        let mut sketch = Sketch2D::default();
        sketch.add_point(SketchEntityId(1), [-scale, 0.0]).unwrap();
        sketch.add_point(SketchEntityId(2), [scale, 0.0]).unwrap();
        sketch
            .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
            .unwrap();
        sketch.set_line_centerline(SketchEntityId(3), true).unwrap();
        assert_eq!(
            sketch.centerline_axis().unwrap().unwrap().id,
            SketchEntityId(3)
        );
    }
}
fn assert_same_numeric_evidence(
    actual: &oblikovati_model::sketch::SketchSolveOutcome,
    expected: &oblikovati_model::sketch::SketchSolveOutcome,
) {
    let actual = actual.numerical();
    let expected = expected.numerical();
    assert_eq!(actual.values, expected.values);
    assert_eq!(actual.termination, expected.termination);
    assert_eq!(
        actual.structure.degrees_of_freedom,
        expected.structure.degrees_of_freedom
    );
}
