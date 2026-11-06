// SPDX-License-Identifier: GPL-2.0-only
use super::arcs::driven_arc;
use super::*;
#[test]
fn mixed_concentric_equal_radius_and_supporting_circle_incidence_share_arc_frames() {
    let mut sketch = driven_arc(1.0, true);
    sketch.add_point(entity(5), [2.0, 3.0]).unwrap();
    sketch.add_circle(entity(6), entity(5), 3.0).unwrap();
    sketch.add_point(entity(7), [6.0, 8.0]).unwrap();
    relation(
        &mut sketch,
        5,
        PlanarConstraint::Concentric {
            curves: [entity(4), entity(6)],
        },
    );
    relation(
        &mut sketch,
        6,
        PlanarConstraint::EqualRadius {
            curves: [entity(6), entity(4)],
        },
    );
    relation(
        &mut sketch,
        7,
        PlanarConstraint::PointOnCircularCurve {
            point: entity(7),
            curve: entity(4),
        },
    );
    relation(
        &mut sketch,
        8,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(7)],
        },
    );
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(result.numerical().structure.rank, 11);
    assert_point(result.candidate(), 5, [0.0, 0.0]);
    assert_point(result.candidate(), 7, [5.0, 0.0]);
    assert!((result.candidate().circle(entity(6)).unwrap().radius - 5.0).abs() < 1e-8);
}
fn free_mixed_contact(internal: bool, reversed: bool, scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    let x = if internal { 1.0 } else { 6.0 } * scale;
    for (id, position) in [
        (1, [0.0, 0.0]),
        (2, [x, 0.2 * scale]),
        (3, [x + 3.0 * scale, 0.2 * scale]),
        (4, [x, 3.2 * scale]),
    ] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch
        .add_circle(entity(5), entity(1), 2.0 * scale)
        .unwrap();
    sketch
        .add_arc(entity(6), entity(2), [entity(3), entity(4)], false)
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
        PlanarConstraint::Radius {
            curve: entity(5),
            parameter: 2,
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Horizontal {
            points: [entity(2), entity(3)],
        },
    );
    relation(
        &mut sketch,
        5,
        PlanarConstraint::Vertical {
            points: [entity(2), entity(4)],
        },
    );
    relation(
        &mut sketch,
        6,
        PlanarConstraint::Radius {
            curve: entity(6),
            parameter: 1,
        },
    );
    let curves = if reversed {
        [entity(6), entity(5)]
    } else {
        [entity(5), entity(6)]
    };
    sketch.add_circular_tangent(constraint(7), curves).unwrap();
    sketch
}
#[test]
fn free_circle_arc_contact_solves_all_enclosure_orders_and_retains_radius_edits() {
    for internal in [false, true] {
        for reversed in [false, true] {
            let mut candidate = free_mixed_contact(internal, reversed, 1.0);
            let original = candidate.clone();
            let mut parameters = parameters();
            parameters.set_expression(2, "2 cm").unwrap();
            for radius in [5.0, 8.0] {
                parameters
                    .set_expression(1, &format!("{radius} cm"))
                    .unwrap();
                let result = candidate
                    .solve(&parameters, SolveOptions::default(), || false)
                    .unwrap();
                assert_eq!(result.numerical().termination, SolveTermination::Converged);
                assert_eq!(result.numerical().structure.rank, 9);
                assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
                let x = if internal { radius - 2.0 } else { radius + 2.0 };
                assert_point(result.candidate(), 2, [x, 0.0]);
                assert_point(result.candidate(), 3, [x + radius, 0.0]);
                assert_point(result.candidate(), 4, [x, radius]);
                assert_eq!(result.candidate().arcs(), original.arcs());
                assert_eq!(result.candidate().constraints(), original.constraints());
                assert_eq!(result.parameter_dependencies(), [2, 1]);
                candidate = result.into_candidate();
            }
        }
    }
}
#[test]
fn mixed_free_contact_and_rank_are_stable_across_working_unit_scales() {
    for scale in [1e-100, 1.0, 1e100] {
        let sketch = free_mixed_contact(false, true, scale);
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        parameters
            .set_expression(2, &format!("{} cm", 2.0 * scale))
            .unwrap();
        assert_eq!(sketch.analyze(&parameters, 1e-9).unwrap().rank, 9);
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert!(
            (result.candidate().point(entity(2)).unwrap().position[0] / scale - 7.0).abs() < 1e-8
        );
    }
}
#[test]
fn circular_relations_reject_unknown_or_non_circular_entities_atomically() {
    let mut sketch = driven_arc(1.0, true);
    let original = sketch.clone();
    for curve in [entity(1), entity(999)] {
        for relation in [
            PlanarConstraint::Radius {
                curve,
                parameter: 1,
            },
            PlanarConstraint::PointOnCircularCurve {
                point: entity(1),
                curve,
            },
            PlanarConstraint::EqualRadius {
                curves: [entity(4), curve],
            },
            PlanarConstraint::Concentric {
                curves: [curve, entity(4)],
            },
        ] {
            assert!(
                sketch
                    .add_constraint(constraint(99), relation)
                    .unwrap_err()
                    .to_string()
                    .contains(&format!("{curve:?}"))
            );
        }
        assert!(
            sketch
                .add_circular_tangent(constraint(99), [entity(4), curve])
                .is_err()
        );
    }
    assert_eq!(sketch, original);
}
#[test]
fn collapsed_derived_radius_is_reported_by_the_authored_radius_constraint() {
    let mut sketch = driven_arc(1.0, true);
    sketch.set_position(entity(2), [0.0, 0.0]).unwrap();
    let error = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(error, SketchError::Residual { constraint: id, .. } if id == constraint(4)));
}

#[test]
fn free_line_arc_contact_drives_derived_radius_and_center_after_parameter_edits() {
    let mut sketch = Sketch2D::default();
    for (id, p) in [
        (1, [0.3, 0.2]),
        (2, [3.3, 0.2]),
        (3, [0.3, -2.8]),
        (5, [-2.0, 5.0]),
        (6, [2.0, 5.0]),
        (8, [0.0, 0.0]),
    ] {
        sketch.add_point(entity(id), p).unwrap();
    }
    sketch
        .add_arc(entity(4), entity(1), [entity(2), entity(3)], true)
        .unwrap();
    sketch.add_line(entity(7), [entity(5), entity(6)]).unwrap();
    for (id, point, position) in [(1, 5, [-2.0, 5.0]), (2, 6, [2.0, 5.0]), (3, 8, [0.0, 0.0])] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(point),
                position,
            },
        );
    }
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(8), entity(1)],
        },
    );
    relation(
        &mut sketch,
        5,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        6,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    relation(
        &mut sketch,
        7,
        PlanarConstraint::Radius {
            curve: entity(4),
            parameter: 1,
        },
    );
    sketch
        .add_line_tangent(constraint(8), entity(7), entity(4))
        .unwrap();
    let original = sketch.clone();
    let mut parameters = parameters();
    for radius in [5.0, 8.0] {
        parameters
            .set_expression(1, &format!("{radius} cm"))
            .unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.rank, 12);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert_point(result.candidate(), 1, [0.0, 5.0 - radius]);
        assert_point(result.candidate(), 2, [radius, 5.0 - radius]);
        assert_point(result.candidate(), 3, [0.0, 5.0 - 2.0 * radius]);
        assert_eq!(result.candidate().constraints(), original.constraints());
        sketch = result.into_candidate();
    }
}
