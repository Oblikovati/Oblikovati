// SPDX-License-Identifier: GPL-2.0-only
//! Conditional affine rows must derive from the sketch equations.
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchAffineError, SketchConstraintId,
        SketchEntityId, SketchIntervalTarget, SketchRelationId,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::affine_preference::{AffineLimits, AffinePreferenceError};

fn horizontal_sketch() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch.add_point(SketchEntityId(2), [4., 0.]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: SketchEntityId(1),
                position: [0., 0.],
            },
        )
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Horizontal {
                points: [SketchEntityId(1), SketchEntityId(2)],
            },
        )
        .unwrap();
    sketch
}

#[test]
fn shared_interval_equations_supply_a_conditional_best_fit_system() {
    let sketch = horizontal_sketch();
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let domain = [[-1., 1.], [-1., 1.], [0., 8.], [-4., 4.]];
    let system = sketch
        .conditional_affine_system(&parameters, &domain, &[0., 0., 4., 0.], &[])
        .unwrap();
    assert_eq!(system.coefficients().len(), 3);
    assert_eq!(system.bounds(), domain);
    let result = system
        .certify_optimum(
            &[(2, 6.), (3, 3.)],
            AffineLimits {
                max_order: 16,
                max_bits: 4096,
            },
            || false,
        )
        .unwrap();
    let coordinates = result.coordinates();
    assert_eq!(coordinates[2].bounds(), [6., 6.]);
    assert_eq!(coordinates[3].bounds(), [0., 0.]);
}

#[test]
fn optimum_outside_the_proved_domain_and_cancellation_are_refused() {
    let sketch = horizontal_sketch();
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let domain = [[-1., 1.], [-1., 1.], [0., 8.], [-4., 4.]];
    let system = sketch
        .conditional_affine_system(&parameters, &domain, &[0., 0., 4., 0.], &[])
        .unwrap();
    let limits = AffineLimits {
        max_order: 16,
        max_bits: 4096,
    };
    assert!(matches!(
        system.certify_optimum(&[(2, 20.)], limits, || false),
        Err(SketchAffineError::OutsideDomain { .. })
    ));
    assert!(matches!(
        system.certify_optimum(&[(2, 6.)], limits, || true),
        Err(SketchAffineError::Solve(AffinePreferenceError::Cancelled))
    ));
}

#[test]
fn anchor_outside_the_domain_is_refused() {
    let sketch = horizontal_sketch();
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let domain = [[-1., 1.], [-1., 1.], [0., 3.], [-4., 4.]];
    assert!(matches!(
        sketch.conditional_affine_system(&parameters, &domain, &[0., 0., 4., 0.], &[]),
        Err(SketchAffineError::Input(_))
    ));
}

#[test]
fn unsupported_nonlinear_relation_is_not_treated_as_affine() {
    let mut sketch = horizontal_sketch();
    sketch.add_point(SketchEntityId(3), [2., 1.]).unwrap();
    sketch
        .add_line(SketchEntityId(4), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(3),
            PlanarConstraint::PointOnLine {
                point: SketchEntityId(3),
                line: SketchEntityId(4),
            },
        )
        .unwrap();
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let domain = [
        [-1., 1.],
        [-1., 1.],
        [0., 8.],
        [-4., 4.],
        [0., 4.],
        [0., 2.],
    ];
    assert!(matches!(
        sketch.conditional_affine_system(&parameters, &domain, &[0., 0., 4., 0., 2., 1.], &[]),
        Err(SketchAffineError::Interval(_))
    ));
}

fn distance_sketch() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [5., 0.]).unwrap();
    sketch.add_point(SketchEntityId(2), [0., 0.]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Distance {
                points: [SketchEntityId(1), SketchEntityId(2)],
                measure: DistanceMeasure::Horizontal,
                parameter: 1,
            },
        )
        .unwrap();
    sketch
}

#[test]
fn distance_requires_a_fixed_target_and_a_known_sign() {
    let sketch = distance_sketch();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "50 mm")
        .unwrap();
    let target = |bounds| SketchIntervalTarget {
        relation: SketchRelationId::Authored(SketchConstraintId(1)),
        parameter: 1,
        bounds,
    };
    let anchor = [5., 0., 0., 0.];
    let positive = [[4., 6.], [0., 0.], [0., 0.], [0., 0.]];
    sketch
        .conditional_affine_system(&parameters, &positive, &anchor, &[target([5., 5.])])
        .unwrap();
    assert!(matches!(
        sketch.conditional_affine_system(&parameters, &positive, &anchor, &[target([4.9, 5.1])]),
        Err(SketchAffineError::Indeterminate { .. })
    ));
    let crossing = [[-1., 6.], [0., 0.], [0., 0.], [0., 0.]];
    assert!(matches!(
        sketch.conditional_affine_system(&parameters, &crossing, &anchor, &[target([5., 5.])]),
        Err(SketchAffineError::Interval(_))
    ));
}

#[test]
fn inexact_anchor_residual_is_not_assumed_exact() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [1., 0.]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: SketchEntityId(1),
                position: [0.1, 0.],
            },
        )
        .unwrap();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    assert!(matches!(
        sketch.conditional_affine_system(&parameters, &[[0., 2.], [0., 0.]], &[1., 0.], &[]),
        Err(SketchAffineError::Indeterminate { .. })
    ));
}
