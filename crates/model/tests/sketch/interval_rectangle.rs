// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use num_rational::BigRational;
use oblikovati_model::sketch::{
    SketchIntervalError, SketchIntervalTarget, SketchPoint2D, SketchRelationId,
};

fn q(value: f64) -> BigRational {
    BigRational::from_float(value).unwrap()
}
fn target(id: u64, parameter: u64, bounds: [f64; 2]) -> SketchIntervalTarget {
    SketchIntervalTarget {
        relation: SketchRelationId::Authored(constraint(id)),
        parameter,
        bounds,
    }
}
fn row(
    rows: &[oblikovati_model::sketch::SketchIntervalEquation],
    id: u64,
    equation: usize,
) -> &oblikovati_model::sketch::SketchIntervalEquation {
    rows.iter()
        .find(|row| {
            row.relation == SketchRelationId::Authored(constraint(id)) && row.equation == equation
        })
        .unwrap()
}
fn rectangle_box() -> [[f64; 2]; 8] {
    [
        [0., 0.],
        [0., 0.],
        [5., 5.],
        [0., 0.],
        [5., 5.],
        [2.5, 2.5],
        [0., 0.],
        [2.5, 2.5],
    ]
}
#[test]
fn rectangle_rows_enclose_uncertain_targets_and_fixed_coordinates() {
    let sketch = rectangle();
    let targets = [target(6, 1, [4.9, 5.1]), target(7, 2, [2.4, 2.6])];
    let rows = sketch
        .interval_equations_with_targets(&parameters(), &rectangle_box(), &targets)
        .unwrap();
    let width = row(&rows, 6, 0);
    assert!(q(width.value[0]) <= q(5.) - q(5.1));
    assert!(q(width.value[1]) >= q(5.) - q(4.9));
    assert_eq!(width.jacobian[0], [-1., -1.]);
    assert_eq!(width.jacobian[2], [1., 1.]);
    let height = row(&rows, 7, 0);
    assert!(q(height.value[0]) <= q(2.5) - q(2.6));
    assert!(q(height.value[1]) >= q(2.5) - q(2.4));
    assert_eq!(row(&rows, 1, 0).value, [0., 0.]);
    assert_eq!(row(&rows, 1, 1).jacobian[1], [1., 1.]);
}
#[test]
fn dimensional_target_identity_bounds_and_branch_are_required() {
    let sketch = rectangle();
    let values = rectangle_box();
    let targets = [target(6, 1, [5., 5.]), target(7, 2, [2.5, 2.5])];
    assert!(matches!(
        sketch.interval_equations(&parameters(), &values),
        Err(SketchIntervalError::Input(_))
    ));
    assert!(matches!(
        sketch.interval_equations_with_targets(
            &parameters(),
            &values,
            &[target(6, 2, [5., 5.]), targets[1].clone()]
        ),
        Err(SketchIntervalError::Input(_))
    ));
    assert!(matches!(
        sketch.interval_equations_with_targets(
            &parameters(),
            &values,
            &[target(6, 1, [6., 7.]), targets[1].clone()]
        ),
        Err(SketchIntervalError::Input(_))
    ));
    assert!(matches!(
        sketch.interval_equations_with_targets(
            &parameters(),
            &values,
            &[targets[0].clone(), targets[0].clone(), targets[1].clone()]
        ),
        Err(SketchIntervalError::Input(_))
    ));
    let mut crossing = values;
    crossing[2] = [-1., 1.];
    assert!(matches!(
        sketch.interval_equations_with_targets(&parameters(), &crossing, &targets),
        Err(SketchIntervalError::Arithmetic { .. })
    ));
}
#[test]
fn midpoint_and_ground_use_exact_authored_targets_and_interval_arithmetic() {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0., 0.]), (2, [2., 0.]), (3, [1., 0.])] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Ground {
            points: vec![SketchPoint2D {
                id: entity(1),
                position: [0.1, -0.2],
            }],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Midpoint {
            point: entity(3),
            line: entity(4),
        },
    );
    let values = [
        [0.1, 0.2],
        [-0.2, -0.1],
        [2., 3.],
        [0., 1.],
        [1., 2.],
        [0., 1.],
    ];
    let rows = sketch.interval_equations(&parameters(), &values).unwrap();
    let fixed = row(&rows, 1, 0);
    assert!(q(fixed.value[0]) <= q(0.1) - q(0.1));
    assert!(q(fixed.value[1]) >= q(0.2) - q(0.1));
    let midpoint = row(&rows, 2, 0);
    let two = q(2.);
    assert!(q(midpoint.value[0]) <= q(1.) - (q(0.2) + q(3.)) / &two);
    assert!(q(midpoint.value[1]) >= q(2.) - (q(0.1) + q(2.)) / &two);
    assert_eq!(midpoint.jacobian[4], [1., 1.]);
    assert_eq!(midpoint.jacobian[0], [-0.5, -0.5]);
    assert_eq!(midpoint.jacobian[2], [-0.5, -0.5]);
}

#[test]
fn positive_axis_distance_has_correct_derivative() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [5., 0.]).unwrap();
    sketch.add_point(entity(2), [0., 0.]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
    let inputs = [[5., 5.], [0., 0.], [0., 0.], [0., 0.]];
    let rows = sketch
        .interval_equations_with_targets(&parameters(), &inputs, &[target(1, 1, [5., 5.])])
        .unwrap();
    assert_eq!(rows[0].value, [0., 0.]);
    assert_eq!(rows[0].jacobian, [[1., 1.], [0., 0.], [-1., -1.], [0., 0.]]);
}
#[test]
fn aligned_distance_is_not_yet_supported() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [5., 0.]).unwrap();
    sketch.add_point(entity(2), [0., 0.]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: DistanceMeasure::Aligned,
            parameter: 1,
        },
    );
    let inputs = [[5., 5.], [0., 0.], [0., 0.], [0., 0.]];
    assert!(matches!(
        sketch.interval_equations_with_targets(&parameters(), &inputs, &[target(1, 1, [5., 5.])]),
        Err(SketchIntervalError::Unsupported(_))
    ));
}
