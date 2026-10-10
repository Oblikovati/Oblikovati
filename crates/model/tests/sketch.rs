// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchError,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::{
    diagnostics::ConstraintStatus,
    nonlinear::{SolveOptions, SolveTermination},
};
fn entity(value: u64) -> SketchEntityId {
    SketchEntityId(value)
}
fn constraint(value: u64) -> SketchConstraintId {
    SketchConstraintId(value)
}
fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "50 mm")
        .unwrap();
    parameters
        .insert_expression(2, "height", USER_PARAM, "width / 2")
        .unwrap();
    parameters
}
fn relation(sketch: &mut Sketch2D, id: u64, relation: PlanarConstraint) {
    sketch.add_constraint(constraint(id), relation).unwrap();
}
fn rectangle() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (index, position) in [[0.1, -0.2], [4.5, 0.3], [5.1, 2.7], [-0.2, 3.2]]
        .into_iter()
        .enumerate()
    {
        sketch
            .add_point(entity(index as u64 + 1), position)
            .unwrap();
    }
    for (id, a, b) in [(5, 1, 2), (6, 2, 3), (7, 3, 4), (8, 4, 1)] {
        sketch.add_line(entity(id), [entity(a), entity(b)]).unwrap();
    }
    rectangle_constraints(&mut sketch);
    sketch
}
fn rectangle_constraints(sketch: &mut Sketch2D) {
    relation(
        sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    for (id, a, b) in [(2, 1, 2), (3, 3, 4)] {
        relation(
            sketch,
            id,
            PlanarConstraint::Horizontal {
                points: [entity(a), entity(b)],
            },
        );
    }
    for (id, a, b) in [(4, 2, 3), (5, 4, 1)] {
        relation(
            sketch,
            id,
            PlanarConstraint::Vertical {
                points: [entity(a), entity(b)],
            },
        );
    }
    relation(
        sketch,
        6,
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
    relation(
        sketch,
        7,
        PlanarConstraint::Distance {
            points: [entity(1), entity(4)],
            measure: DistanceMeasure::Vertical,
            parameter: 2,
        },
    );
}
fn assert_point(sketch: &Sketch2D, id: u64, expected: [f64; 2]) {
    for (actual, expected) in sketch
        .point(entity(id))
        .unwrap()
        .position
        .into_iter()
        .zip(expected)
    {
        assert!(
            (actual - expected).abs() < 1e-8,
            "actual {actual}, expected {expected}"
        );
    }
}
fn assert_rectangle(sketch: &Sketch2D, width: f64, height: f64) {
    for (id, position) in [
        (1, [0.0, 0.0]),
        (2, [width, 0.0]),
        (3, [width, height]),
        (4, [0.0, height]),
    ] {
        assert_point(sketch, id, position);
    }
}
#[test]
fn parameter_driven_rectangle_is_fully_constrained_and_recomputes_after_formula_edit() {
    let sketch = rectangle();
    let original = sketch.clone();
    let mut parameters = parameters();
    let outcome = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().termination, SolveTermination::Converged);
    assert_eq!(
        outcome.numerical().structure.status,
        ConstraintStatus::WellConstrained
    );
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(outcome.numerical().structure.rank, 8);
    assert_eq!(outcome.parameter_dependencies(), [1, 2]);
    assert_rectangle(outcome.candidate(), 5.0, 2.5);
    assert_eq!(sketch, original);
    assert_eq!(outcome.candidate().constraints(), original.constraints());
    assert_eq!(outcome.candidate().lines(), original.lines());
    parameters.set_expression(1, "10 cm").unwrap();
    let next = outcome
        .candidate()
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(next.numerical().termination, SolveTermination::Converged);
    assert_rectangle(next.candidate(), 10.0, 5.0);
    assert_eq!(next.into_candidate().constraints(), original.constraints());
}
#[test]
fn all_owned_points_count_as_dof_and_shared_line_endpoints_add_no_duplicate_variables() {
    let mut sketch = rectangle();
    let parameters = parameters();
    sketch.add_point(entity(9), [19.0, 21.0]).unwrap();
    assert_eq!(
        sketch
            .analyze(&parameters, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        2
    );
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().structure.variables, 10);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 2);
    assert_point(result.candidate(), 9, [19.0, 21.0]);
    assert_eq!(
        sketch.line(entity(5)).unwrap().endpoints[1],
        sketch.line(entity(6)).unwrap().endpoints[0]
    );
    let mut free = Sketch2D::default();
    free.add_point(entity(4), [1.0, 2.0]).unwrap();
    assert_eq!(
        free.analyze(&parameters, 1e-9).unwrap().degrees_of_freedom,
        2
    );
    assert_eq!(
        Sketch2D::default()
            .analyze(&parameters, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        0
    );
}
#[test]
fn duplicate_and_conflicting_relations_report_stable_constraint_identities() {
    let mut sketch = rectangle();
    relation(
        &mut sketch,
        99,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    let parameters = parameters();
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(
        result.numerical().structure.status,
        ConstraintStatus::OverConstrained
    );
    assert!(result.conflicting_constraints().is_empty());
    assert_eq!(
        result.redundant_constraints(1e-9).unwrap(),
        [constraint(2), constraint(99)]
    );
    assert!(result.redundant_constraints(f64::NAN).is_err());
    let mut conflict = Sketch2D::default();
    conflict.add_point(entity(1), [0.5, 0.0]).unwrap();
    for (id, position) in [(71, [0.0, 0.0]), (27, [1.0, 0.0])] {
        relation(
            &mut conflict,
            id,
            PlanarConstraint::Fix {
                point: entity(1),
                position,
            },
        );
    }
    let result = conflict
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Stalled);
    assert_eq!(
        result.conflicting_constraints(),
        [constraint(71), constraint(27)]
    );
    assert_point(&conflict, 1, [0.5, 0.0]);
}
#[test]
fn midpoint_and_coincidence_share_variable_identity_without_reseeding_local_occurrences() {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(20, [0.0, 0.0]), (10, [4.0, 6.0]), (30, [1.0, 2.0])] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch
        .add_line(entity(40), [entity(20), entity(10)])
        .unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(20),
            position: [0.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Fix {
            point: entity(10),
            position: [4.0, 6.0],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Midpoint {
            point: entity(30),
            line: entity(40),
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_point(result.candidate(), 30, [2.0, 3.0]);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Coincident {
            points: [entity(30), entity(30)],
        },
    );
    assert_eq!(
        sketch
            .analyze(&parameters(), 1e-9)
            .unwrap()
            .redundant_equations,
        2
    );
    let mut repeated = Sketch2D::default();
    repeated.add_point(entity(1), [0.0, 0.0]).unwrap();
    repeated.add_point(entity(2), [4.0, 6.0]).unwrap();
    repeated
        .add_line(entity(3), [entity(1), entity(2)])
        .unwrap();
    relation(
        &mut repeated,
        1,
        PlanarConstraint::Midpoint {
            point: entity(1),
            line: entity(3),
        },
    );
    assert_eq!(repeated.analyze(&parameters(), 1e-9).unwrap().rank, 2);
}
#[test]
fn geometry_and_reference_errors_are_atomic_and_include_offending_values() {
    let mut sketch = rectangle();
    let original = sketch.clone();
    for (id, position) in [
        (0, [0.0, 0.0]),
        (1, [0.0, 0.0]),
        (5, [0.0, 0.0]),
        (50, [f64::NAN, 0.0]),
    ] {
        assert!(sketch.add_point(entity(id), position).is_err());
    }
    assert!(
        sketch
            .add_line(entity(50), [entity(1), entity(999)])
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert!(sketch.set_position(entity(999), [0.0, 0.0]).is_err());
    assert!(
        sketch
            .set_position(entity(1), [f64::INFINITY, 0.0])
            .is_err()
    );
    assert!(sketch.line(entity(1)).is_err());
    assert!(
        sketch
            .add_constraint(
                constraint(0),
                PlanarConstraint::Horizontal {
                    points: [entity(1), entity(2)]
                }
            )
            .is_err()
    );
    assert!(
        sketch
            .add_constraint(
                constraint(1),
                PlanarConstraint::Horizontal {
                    points: [entity(1), entity(2)]
                }
            )
            .is_err()
    );
    assert!(
        sketch
            .add_constraint(
                constraint(50),
                PlanarConstraint::Midpoint {
                    point: entity(1),
                    line: entity(999)
                }
            )
            .is_err()
    );
    assert!(
        sketch
            .add_constraint(
                constraint(50),
                PlanarConstraint::Fix {
                    point: entity(1),
                    position: [f64::NAN, 0.0]
                }
            )
            .is_err()
    );
    assert_eq!(sketch, original);
    sketch.set_position(entity(1), [2.0, 3.0]).unwrap();
    assert_point(&sketch, 1, [2.0, 3.0]);
}
#[test]
fn distance_targets_reject_unhealthy_wrong_unit_and_negative_parameters() {
    let sketch = rectangle();
    let mut parameters = parameters();
    parameters.set_expression(1, "3 rad").unwrap();
    assert!(
        sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap_err()
            .to_string()
            .contains("working-unit length")
    );
    parameters.set_expression(1, "-3 cm").unwrap();
    assert!(
        sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .is_err()
    );
    parameters.set_expression(1, "unknown_length").unwrap();
    assert!(matches!(
        sketch.solve(&parameters, SolveOptions::default(), || false),
        Err(SketchError::Parameter { parameter: 1, .. })
    ));
    assert!(
        sketch
            .solve(
                &ParameterCollection::new(UnitsOfMeasure::default()),
                SolveOptions::default(),
                || false
            )
            .unwrap_err()
            .to_string()
            .contains("parameter 1")
    );
}
#[test]
fn cancellation_and_degenerate_distance_do_not_mutate_authored_sketch() {
    let sketch = rectangle();
    let cancelled = sketch
        .solve(&parameters(), SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(
        cancelled.numerical().termination,
        SolveTermination::Cancelled
    );
    assert_eq!(cancelled.candidate(), &sketch);
    let first = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let second = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(first.candidate(), second.candidate());
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Distance {
            points: [entity(1), entity(1)],
            measure: DistanceMeasure::Aligned,
            parameter: 1,
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Stalled);
    assert_eq!(result.conflicting_constraints(), [constraint(1)]);
    assert_eq!(result.parameter_dependencies(), [1]);
}

#[path = "sketch/dimensions.rs"]
mod dimensions;

#[path = "sketch/lines.rs"]
mod lines;

#[path = "sketch/circles.rs"]
mod circles;

#[path = "sketch/tangency.rs"]
mod tangency;

#[path = "sketch/arcs.rs"]
mod arcs;

#[path = "sketch/circular_constraints.rs"]
mod circular_constraints;

#[path = "sketch/shared_contact.rs"]
mod shared_contact;

#[path = "sketch/arc_midpoints.rs"]
mod arc_midpoints;

#[path = "sketch/angles.rs"]
mod angles;

#[path = "sketch/circular_dimensions.rs"]
mod circular_dimensions;

#[path = "sketch/tangent_distances.rs"]
mod tangent_distances;
