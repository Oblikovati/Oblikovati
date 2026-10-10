// SPDX-License-Identifier: GPL-2.0-only
use super::arcs::driven_arc;
use super::*;
use std::f64::consts::PI;
fn dimensioned_arc(
    counter_clockwise: bool,
    scale: f64,
    relation_value: PlanarConstraint,
) -> Sketch2D {
    let seed = driven_arc(scale, counter_clockwise);
    let mut sketch = Sketch2D::default();
    for point in seed.points() {
        sketch.add_point(point.id, point.position).unwrap();
    }
    sketch
        .add_arc(
            entity(4),
            entity(1),
            [entity(2), entity(3)],
            counter_clockwise,
        )
        .unwrap();
    for constraint in &seed.constraints()[..3] {
        sketch
            .add_constraint(constraint.id, constraint.relation.clone())
            .unwrap();
    }
    relation(&mut sketch, 4, relation_value);
    sketch
}
#[test]
fn directed_arc_lengths_drive_minor_and_major_sweeps_with_zero_dof_after_formula_edits() {
    for counter_clockwise in [false, true] {
        let original = dimensioned_arc(
            counter_clockwise,
            1.0,
            PlanarConstraint::ArcLength {
                arc: entity(4),
                parameter: 3,
            },
        );
        let mut candidate = original.clone();
        let mut parameters = parameters();
        let sweep = if counter_clockwise {
            PI / 2.0
        } else {
            3.0 * PI / 2.0
        };
        parameters
            .insert_expression(3, "arc_path", USER_PARAM, &format!("width * {sweep}"))
            .unwrap();
        for radius in [5.0, 8.0] {
            parameters
                .set_expression(1, &format!("{radius} cm"))
                .unwrap();
            let result = candidate
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
            assert_eq!(result.numerical().structure.rank, 6);
            assert_point(result.candidate(), 2, [radius, 0.0]);
            assert_point(result.candidate(), 3, [0.0, radius]);
            assert_eq!(result.candidate().arcs(), original.arcs());
            assert_eq!(result.candidate().constraints(), original.constraints());
            assert_eq!(result.parameter_dependencies(), [3]);
            candidate = result.into_candidate();
        }
    }
}
#[test]
fn diameter_dimensions_drive_circle_and_arc_radii_from_one_common_equation() {
    let mut parameters = parameters();
    parameters
        .insert_expression(3, "diameter", USER_PARAM, "width * 2")
        .unwrap();
    let mut arc = dimensioned_arc(
        true,
        1.0,
        PlanarConstraint::Diameter {
            curve: entity(4),
            parameter: 3,
        },
    );
    let mut circle = Sketch2D::default();
    circle.add_point(entity(1), [0.2, -0.1]).unwrap();
    circle.add_circle(entity(2), entity(1), 3.0).unwrap();
    relation(
        &mut circle,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    relation(
        &mut circle,
        2,
        PlanarConstraint::Diameter {
            curve: entity(2),
            parameter: 3,
        },
    );
    for radius in [5.0, 8.0] {
        parameters
            .set_expression(1, &format!("{radius} cm"))
            .unwrap();
        let result = arc
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert_point(result.candidate(), 2, [radius, 0.0]);
        assert_point(result.candidate(), 3, [0.0, radius]);
        arc = result.into_candidate();
        let result = circle
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.rank, 3);
        assert!((result.candidate().circle(entity(2)).unwrap().radius - radius).abs() < 1e-8);
        assert_eq!(result.parameter_dependencies(), [3]);
        circle = result.into_candidate();
    }
}
#[test]
fn circular_dimension_domains_and_references_are_named_atomic_errors() {
    let mut parameters = parameters();
    parameters
        .insert_expression(3, "dimension", USER_PARAM, "1 cm")
        .unwrap();
    let arc_length = dimensioned_arc(
        true,
        1.0,
        PlanarConstraint::ArcLength {
            arc: entity(4),
            parameter: 3,
        },
    );
    let diameter = dimensioned_arc(
        true,
        1.0,
        PlanarConstraint::Diameter {
            curve: entity(4),
            parameter: 3,
        },
    );
    for expression in ["0 cm", "-1 cm", "1 deg"] {
        parameters.set_expression(3, expression).unwrap();
        for sketch in [&arc_length, &diameter] {
            assert!(
                sketch
                    .solve(&parameters, SolveOptions::default(), || false)
                    .unwrap_err()
                    .to_string()
                    .contains("parameter 3")
            );
        }
    }
    parameters.set_expression(3, "5e-324 cm").unwrap();
    assert!(
        diameter
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap_err()
            .to_string()
            .contains("positive half-diameter")
    );
    let mut sketch = arc_length.clone();
    let original = sketch.clone();
    for relation in [
        PlanarConstraint::Diameter {
            curve: entity(999),
            parameter: 3,
        },
        PlanarConstraint::ArcLength {
            arc: entity(999),
            parameter: 3,
        },
    ] {
        assert!(
            sketch
                .add_constraint(constraint(99), relation)
                .unwrap_err()
                .to_string()
                .contains("999")
        );
    }
    assert_eq!(sketch, original);
    parameters.set_expression(3, "5 cm").unwrap();
    sketch.set_position(entity(3), [3.0, 0.0]).unwrap();
    let error = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(error, SketchError::Residual { constraint: id, .. } if id == constraint(4)));
}
#[test]
fn directed_length_and_diameter_solve_rank_and_cancellation_preserve_working_units() {
    for scale in [1e-100, 1.0, 1e100] {
        for counter_clockwise in [false, true] {
            let mut parameters = parameters();
            let sweep = if counter_clockwise {
                PI / 2.0
            } else {
                3.0 * PI / 2.0
            };
            parameters
                .set_expression(1, &format!("{} cm", 5.0 * scale))
                .unwrap();
            parameters
                .insert_expression(3, "arc_path", USER_PARAM, &format!("width * {sweep}"))
                .unwrap();
            for constraint in [
                PlanarConstraint::ArcLength {
                    arc: entity(4),
                    parameter: 3,
                },
                PlanarConstraint::Diameter {
                    curve: entity(4),
                    parameter: 1,
                },
            ] {
                let is_diameter = matches!(constraint, PlanarConstraint::Diameter { .. });
                let sketch = dimensioned_arc(counter_clockwise, scale, constraint);
                let result = sketch
                    .solve(&parameters, SolveOptions::default(), || false)
                    .unwrap();
                assert_eq!(result.numerical().termination, SolveTermination::Converged);
                assert_eq!(result.numerical().structure.rank, 6);
                assert!(
                    (result.candidate().arc_radius(entity(4)).unwrap() / scale
                        - if is_diameter { 2.5 } else { 5.0 })
                    .abs()
                        < 1e-8
                );
                let cancelled = sketch
                    .solve(&parameters, SolveOptions::default(), || true)
                    .unwrap();
                assert_eq!(
                    cancelled.numerical().termination,
                    SolveTermination::Cancelled
                );
                assert_eq!(cancelled.candidate(), &sketch);
            }
        }
    }
}
