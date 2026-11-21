// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
fn aligned(axis: [f64; 2], radii: [f64; 2], major_axis: bool, vertical: bool) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [2.0, -3.0]).unwrap();
    sketch.add_ellipse(ellipse(axis, radii)).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::EllipseAxisAlignment {
                ellipse: SketchEntityId(2),
                major_axis,
                vertical,
            },
        )
        .unwrap();
    sketch
}
#[test]
fn alignment_solves_tilted_opposite_and_perpendicular_seeds_for_both_axes() {
    for major in [false, true] {
        for vertical in [false, true] {
            for axis in [[3.0, 4.0], [-3.0, -4.0], [1.0, 0.0], [0.0, 1.0]] {
                assert_axis_solution(
                    &aligned(axis, [3.0, 1.0], major, vertical),
                    major == vertical,
                );
            }
        }
    }
}
fn assert_axis_solution(sketch: &Sketch2D, major_vertical: bool) {
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 4);
    let value = outcome.candidate().ellipse(SketchEntityId(2)).unwrap();
    assert!(
        value.major_axis()[usize::from(!major_vertical)].abs() < 1e-8,
        "{value:?} {:?}",
        outcome.numerical()
    );
    assert_eq!(
        value.radii(),
        sketch.ellipse(SketchEntityId(2)).unwrap().radii()
    );
    assert_fixed_components(&outcome);
}
fn assert_fixed_components(outcome: &oblikovati_model::sketch::SketchSolveOutcome) {
    assert_eq!(
        outcome
            .candidate()
            .point(SketchEntityId(1))
            .unwrap()
            .position,
        [2.0, -3.0]
    );
    let jacobian = &outcome.numerical().evaluation.jacobian()[0];
    assert_eq!(&jacobian[..4], &[0.0; 4]);
    assert!((jacobian[4] - 1.0).abs() < 1e-14);
}
#[test]
fn thin_and_equal_radius_ellipses_keep_frame_identity_and_round_trip_alignment() {
    for radii in [[1.0; 2], [1e6, 1e-6], [1e-6, 1e-12]] {
        let sketch = aligned([3.0, 4.0], radii, true, false);
        assert_axis_solution(&sketch, false);
        assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
    }
}
#[test]
fn ellipse_axis_alignment_refuses_a_nonellipse_owner() {
    let mut sketch = owned_ellipse();
    let original = sketch.recipe();
    let cause = sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::EllipseAxisAlignment {
                ellipse: SketchEntityId(1),
                major_axis: true,
                vertical: false,
            },
        )
        .unwrap_err()
        .to_string();
    assert!(cause.contains("1"));
    assert!(cause.contains("expected"));
    assert_eq!(sketch.recipe(), original);
}
