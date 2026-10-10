// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn circle_sketch(radius: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.2, -0.1]).unwrap();
    sketch.add_circle(entity(2), entity(1), radius).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Radius {
            circle: entity(2),
            parameter: 1,
        },
    );
    sketch
}
fn assert_radius(sketch: &Sketch2D, id: u64, expected: f64) {
    let actual = sketch.circle(entity(id)).unwrap().radius;
    assert!(
        (actual - expected).abs() < 1e-8,
        "radius {actual}, expected {expected}"
    );
}
#[test]
fn parameter_driven_circle_has_zero_dof_and_recomputes_without_replacing_identity() {
    let sketch = circle_sketch(3.0);
    let original = sketch.clone();
    let mut parameters = parameters();
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.rank, 3);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    assert_point(result.candidate(), 1, [0.0, 0.0]);
    assert_radius(result.candidate(), 2, 5.0);
    assert_eq!(result.parameter_dependencies(), [1]);
    assert_eq!(result.candidate().circles()[0].center, entity(1));
    assert_eq!(result.candidate().constraints(), original.constraints());
    assert_eq!(sketch, original);
    parameters.set_expression(1, "8 cm").unwrap();
    let updated = result
        .candidate()
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(updated.numerical().termination, SolveTermination::Converged);
    assert_radius(updated.candidate(), 2, 8.0);
    assert_eq!(updated.candidate().circles()[0].id, entity(2));
}
#[test]
fn concentric_equal_radius_circles_solve_with_independent_radius_variables() {
    let mut sketch = circle_sketch(3.0);
    sketch.add_point(entity(3), [2.0, 3.0]).unwrap();
    sketch.add_circle(entity(4), entity(3), 2.0).unwrap();
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Concentric {
            circles: [entity(2), entity(4)],
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::EqualRadius {
            circles: [entity(2), entity(4)],
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.variables, 6);
    assert_eq!(result.numerical().structure.rank, 6);
    assert_eq!(result.numerical().structure.redundant_equations, 0);
    assert_point(result.candidate(), 3, [0.0, 0.0]);
    assert_radius(result.candidate(), 2, 5.0);
    assert_radius(result.candidate(), 4, 5.0);
    assert_eq!(result.candidate().circles()[1].center, entity(3));
}
#[test]
fn incidence_drives_a_point_to_the_analytic_circle_with_one_free_direction() {
    let mut sketch = circle_sketch(5.0);
    sketch.set_position(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(3), [6.0, 8.0]).unwrap();
    relation(
        &mut sketch,
        3,
        PlanarConstraint::PointOnCircle {
            point: entity(3),
            circle: entity(2),
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 1);
    assert_point(result.candidate(), 3, [3.0, 4.0]);
    assert_radius(result.candidate(), 2, 5.0);
}
#[test]
fn shared_centers_and_free_radii_are_counted_without_duplicate_coordinate_variables() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_circle(entity(2), entity(1), 3.0).unwrap();
    sketch.add_circle(entity(3), entity(1), 7.0).unwrap();
    assert_eq!(
        sketch
            .analyze(&parameters(), 1e-9)
            .unwrap()
            .degrees_of_freedom,
        4
    );
    relation(
        &mut sketch,
        1,
        PlanarConstraint::EqualRadius {
            circles: [entity(2), entity(3)],
        },
    );
    assert_eq!(
        sketch
            .analyze(&parameters(), 1e-9)
            .unwrap()
            .degrees_of_freedom,
        3
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Concentric {
            circles: [entity(2), entity(3)],
        },
    );
    let structure = sketch.analyze(&parameters(), 1e-9).unwrap();
    assert_eq!(structure.variables, 4);
    assert_eq!(structure.redundant_equations, 2);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_radius(result.candidate(), 2, 5.0);
    assert_radius(result.candidate(), 3, 5.0);
    assert_eq!(result.redundant_constraints(1e-9).unwrap(), [constraint(2)]);
}
#[test]
fn circle_identity_radius_and_reference_validation_is_atomic() {
    let mut sketch = circle_sketch(3.0);
    let original = sketch.clone();
    for radius in [0.0, -1.0, f64::NAN, f64::INFINITY] {
        assert!(sketch.add_circle(entity(9), entity(1), radius).is_err());
        assert!(sketch.set_radius(entity(2), radius).is_err());
    }
    for id in [0, 1, 2] {
        assert!(sketch.add_circle(entity(id), entity(1), 1.0).is_err());
    }
    assert!(
        sketch
            .add_circle(entity(9), entity(999), 1.0)
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert!(sketch.add_point(entity(2), [0.0, 0.0]).is_err());
    assert!(sketch.add_line(entity(2), [entity(1), entity(1)]).is_err());
    assert!(sketch.set_radius(entity(999), 1.0).is_err());
    assert!(
        sketch
            .add_constraint(
                constraint(9),
                PlanarConstraint::EqualRadius {
                    circles: [entity(2), entity(999)]
                }
            )
            .is_err()
    );
    assert_eq!(sketch, original);
    sketch.set_radius(entity(2), 4.0).unwrap();
    assert_radius(&sketch, 2, 4.0);
}
#[test]
fn radius_targets_reject_zero_and_failed_solutions_keep_positive_geometry() {
    let sketch = circle_sketch(3.0);
    let mut parameters = parameters();
    parameters.set_expression(1, "0 cm").unwrap();
    assert!(
        sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap_err()
            .to_string()
            .contains("strictly positive")
    );
    parameters.set_expression(1, "-1 cm").unwrap();
    assert!(
        sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .is_err()
    );
    let mut sketch = circle_sketch(3.0);
    relation(
        &mut sketch,
        3,
        PlanarConstraint::PointOnCircle {
            point: entity(1),
            circle: entity(2),
        },
    );
    let result = sketch
        .solve(&super::parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert!(
        result
            .candidate()
            .circles()
            .iter()
            .all(|circle| circle.radius > 0.0)
    );
    assert!(result.conflicting_constraints().contains(&constraint(3)));
    assert_radius(&sketch, 2, 3.0);
    let cancelled = sketch
        .solve(&super::parameters(), SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(
        cancelled.numerical().termination,
        SolveTermination::Cancelled
    );
    assert_eq!(cancelled.candidate(), &sketch);
}
#[test]
fn circle_dof_and_solutions_survive_working_coordinate_scale_sweeps() {
    for scale in [1e-100, 1e-6, 1.0, 1e6, 1e100] {
        let mut sketch = circle_sketch(3.0 * scale);
        sketch
            .set_position(entity(1), [0.2 * scale, -0.1 * scale])
            .unwrap();
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let solved = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(solved.numerical().structure.rank, 3);
        assert!((solved.candidate().circle(entity(2)).unwrap().radius / scale - 5.0).abs() < 1e-8);
    }
}

#[test]
fn trial_steps_that_cross_the_positive_radius_domain_are_rejected() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [10.0, 0.0]).unwrap();
    sketch.add_circle(entity(3), entity(1), 0.1).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Fix {
            point: entity(2),
            position: [-10.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::PointOnCircle {
            point: entity(2),
            circle: entity(3),
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert!(result.numerical().rejected_trial.is_some());
    assert_point(result.candidate(), 2, [-10.0, 0.0]);
    assert_radius(result.candidate(), 3, 10.0);
    assert_radius(&sketch, 3, 0.1);
}

#[test]
fn unconstrained_radii_keep_their_authored_values_while_centers_move() {
    for radius in [1e-200_f64, 1.0, 1e200] {
        let mut sketch = Sketch2D::default();
        let coordinate_scale = radius.max(1.0);
        sketch
            .add_point(entity(1), [3.0 * coordinate_scale, 4.0 * coordinate_scale])
            .unwrap();
        sketch.add_circle(entity(2), entity(1), radius).unwrap();
        relation(
            &mut sketch,
            1,
            PlanarConstraint::Fix {
                point: entity(1),
                position: [0.0, 0.0],
            },
        );
        let result = sketch
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 1);
        assert!(result.numerical().iterations > 0);
        assert_eq!(
            result
                .candidate()
                .circle(entity(2))
                .unwrap()
                .radius
                .to_bits(),
            radius.to_bits()
        );
    }
}
