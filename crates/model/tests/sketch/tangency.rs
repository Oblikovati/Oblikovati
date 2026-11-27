// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{CircleContact, LineContactSide};
use std::error::Error;
fn line_circle(y: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [-4.0, 0.0]), (2, [4.0, 0.0]), (3, [0.0, y])] {
        sketch.add_point(entity(id), position).unwrap();
        if id < 3 {
            relation(
                &mut sketch,
                id,
                PlanarConstraint::Fix {
                    point: entity(id),
                    position,
                },
            );
        }
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    sketch.add_circle(entity(5), entity(3), 3.0).unwrap();
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Radius {
            curve: entity(5),
            parameter: 1,
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(3), entity(1)],
        },
    );
    sketch
}
fn circle_pair(distance: f64, radii: [f64; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [distance, 0.0])] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch.add_circle(entity(3), entity(1), radii[0]).unwrap();
    sketch.add_circle(entity(4), entity(2), radii[1]).unwrap();
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
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Radius {
            curve: entity(3),
            parameter: 1,
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Radius {
            curve: entity(4),
            parameter: 2,
        },
    );
    sketch
}
#[test]
fn line_contact_retains_both_sides_and_recomputes_after_radius_edits() {
    for (y, side, sign) in [
        (2.0, LineContactSide::Left, 1.0),
        (-2.0, LineContactSide::Right, -1.0),
    ] {
        let mut sketch = line_circle(y);
        sketch
            .add_line_tangent(constraint(5), entity(4), entity(5))
            .unwrap();
        assert_eq!(
            sketch.constraints()[4].relation,
            PlanarConstraint::LineTangent {
                line: entity(4),
                curve: entity(5),
                side
            }
        );
        let original = sketch.clone();
        let mut parameters = parameters();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert_point(result.candidate(), 3, [-4.0, sign * 5.0]);
        assert_eq!(sketch, original);
        parameters.set_expression(1, "8 cm").unwrap();
        let next = result
            .candidate()
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_point(next.candidate(), 3, [-4.0, sign * 8.0]);
        assert_eq!(next.candidate().constraints(), original.constraints());
    }
}
#[test]
fn nearest_circle_contact_selection_retains_external_and_enclosing_modes() {
    for (distance, radii, expected_contact, expected_distance) in [
        (7.0, [5.0, 2.5], CircleContact::External, 7.5),
        (2.0, [5.0, 2.5], CircleContact::FirstContainsSecond, 2.5),
        (2.0, [2.5, 5.0], CircleContact::SecondContainsFirst, 2.5),
    ] {
        let mut sketch = circle_pair(distance, radii);
        sketch
            .add_circular_tangent(constraint(5), [entity(3), entity(4)])
            .unwrap();
        assert_eq!(
            sketch.constraints()[4].relation,
            PlanarConstraint::CircularTangent {
                curves: [entity(3), entity(4)],
                contact: expected_contact
            }
        );
        let mut parameters = parameters();
        if expected_contact == CircleContact::SecondContainsFirst {
            parameters.set_expression(2, "5 cm").unwrap();
            parameters.set_expression(1, "2.5 cm").unwrap();
        }
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.rank, 6);
        assert_point(result.candidate(), 2, [expected_distance, 0.0]);
        assert_eq!(result.candidate().constraints(), sketch.constraints());
    }
}
#[test]
fn equal_distance_selection_ties_choose_external_contact() {
    let mut sketch = circle_pair(5.0, [5.0, 2.5]);
    sketch
        .add_circular_tangent(constraint(5), [entity(3), entity(4)])
        .unwrap();
    assert!(matches!(
        sketch.constraints()[4].relation,
        PlanarConstraint::CircularTangent {
            contact: CircleContact::External,
            ..
        }
    ));
}
#[test]
fn invalid_contact_seeds_are_named_and_selection_is_atomic() {
    let mut sketch = line_circle(0.0);
    let original = sketch.clone();
    let error = sketch
        .add_line_tangent(constraint(81), entity(4), entity(5))
        .unwrap_err();
    assert!(error.to_string().contains("81"));
    assert!(error.to_string().contains("nonzero seed side"));
    assert!(error.source().is_some());
    assert_eq!(sketch, original);
    sketch.set_position(entity(2), [-4.0, 0.0]).unwrap();
    assert!(
        sketch
            .add_line_tangent(constraint(81), entity(4), entity(5))
            .unwrap_err()
            .to_string()
            .contains("nonzero direction")
    );
    assert!(
        sketch
            .add_line_tangent(constraint(81), entity(999), entity(5))
            .is_err()
    );
    let mut sketch = circle_pair(0.0, [5.0, 2.5]);
    assert!(
        sketch
            .add_circular_tangent(constraint(82), [entity(3), entity(4)])
            .unwrap_err()
            .to_string()
            .contains("distinct centers")
    );
    assert!(
        sketch
            .add_circular_tangent(constraint(82), [entity(3), entity(999)])
            .is_err()
    );
}
#[test]
fn internal_contact_never_silently_changes_its_enclosing_circle_after_parameter_edit() {
    let mut sketch = circle_pair(2.0, [5.0, 2.5]);
    sketch
        .add_circular_tangent(constraint(5), [entity(3), entity(4)])
        .unwrap();
    let mut parameters = parameters();
    parameters.set_expression(2, "6 cm").unwrap();
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert!(matches!(
        result.candidate().constraints()[4].relation,
        PlanarConstraint::CircularTangent {
            contact: CircleContact::FirstContainsSecond,
            ..
        }
    ));
    assert!(
        result.candidate().circle(entity(3)).unwrap().radius
            > result.candidate().circle(entity(4)).unwrap().radius
    );
    assert!(!result.conflicting_constraints().is_empty());
}
#[test]
fn a_recorded_line_side_does_not_change_when_the_seed_crosses_the_line() {
    let mut sketch = line_circle(2.0);
    sketch
        .add_line_tangent(constraint(5), entity(4), entity(5))
        .unwrap();
    sketch.set_position(entity(3), [0.0, -2.0]).unwrap();
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_point(result.candidate(), 3, [-4.0, 5.0]);
    let cancelled = sketch
        .solve(&parameters(), SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(cancelled.candidate(), &sketch);
}

#[test]
fn contact_side_selection_does_not_lose_small_offsets_next_to_a_large_radius() {
    for (y, side) in [(2.0, LineContactSide::Left), (-2.0, LineContactSide::Right)] {
        let mut sketch = line_circle(y);
        sketch.set_radius(entity(5), 1e200).unwrap();
        sketch
            .add_line_tangent(constraint(5), entity(4), entity(5))
            .unwrap();
        assert!(
            matches!(sketch.constraints()[4].relation, PlanarConstraint::LineTangent { side: actual, .. } if actual == side)
        );
    }
}

#[test]
fn tangent_solve_and_rank_remain_stable_across_working_coordinate_units() {
    for scale in [1e-100, 1e-6, 1.0, 1e6, 1e100] {
        let mut sketch = Sketch2D::default();
        for (id, position) in [(1, [0.0, 0.0]), (2, [7.0 * scale, 0.0])] {
            sketch.add_point(entity(id), position).unwrap();
        }
        sketch
            .add_circle(entity(3), entity(1), 5.0 * scale)
            .unwrap();
        sketch
            .add_circle(entity(4), entity(2), 2.5 * scale)
            .unwrap();
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
            PlanarConstraint::Horizontal {
                points: [entity(1), entity(2)],
            },
        );
        relation(
            &mut sketch,
            3,
            PlanarConstraint::Radius {
                curve: entity(3),
                parameter: 1,
            },
        );
        relation(
            &mut sketch,
            4,
            PlanarConstraint::Radius {
                curve: entity(4),
                parameter: 2,
            },
        );
        sketch
            .add_circular_tangent(constraint(5), [entity(3), entity(4)])
            .unwrap();
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.rank, 6);
        assert!(
            (result.candidate().point(entity(2)).unwrap().position[0] / scale - 7.5).abs() < 1e-8
        );
    }
}
