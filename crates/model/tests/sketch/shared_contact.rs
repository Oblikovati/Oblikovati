// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{CircleContact, LineContactSide};
fn shared_line_arc(at_b: bool, touch_end: bool, far_y: f64, scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, p) in [
        (1, [0.0, 0.0]),
        (2, [0.0, scale]),
        (3, [0.0, -scale]),
        (5, [5.0 * scale, far_y * scale]),
    ] {
        sketch.add_point(entity(id), p).unwrap();
    }
    sketch
        .add_arc(entity(4), entity(1), [entity(2), entity(3)], true)
        .unwrap();
    let touch = if touch_end { entity(3) } else { entity(2) };
    let endpoints = if at_b {
        [entity(5), touch]
    } else {
        [touch, entity(5)]
    };
    sketch.add_line(entity(6), endpoints).unwrap();
    sketch
}
fn constrain_shared_line(sketch: &mut Sketch2D, scale: f64) {
    relation(
        sketch,
        2,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    relation(
        sketch,
        3,
        PlanarConstraint::Fix {
            point: entity(2),
            position: [0.0, scale],
        },
    );
    relation(
        sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    relation(
        sketch,
        5,
        PlanarConstraint::Distance {
            points: [entity(1), entity(5)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
}
#[test]
fn every_shared_line_endpoint_and_arc_endpoint_adds_rank_at_exact_tangency() {
    for at_b in [false, true] {
        for touch_end in [false, true] {
            let far_y = if touch_end { -1.0 } else { 1.0 };
            let mut sketch = shared_line_arc(at_b, touch_end, far_y, 1.0);
            let before = sketch.analyze(&parameters(), 1e-9).unwrap().rank;
            sketch
                .add_line_tangent(constraint(1), entity(6), entity(4))
                .unwrap();
            assert_eq!(
                sketch.analyze(&parameters(), 1e-9).unwrap().rank,
                before + 1
            );
            assert_eq!(before, 1);
        }
    }
}
#[test]
fn pinned_line_contact_converges_from_off_tangent_with_zero_dof_and_preserves_side() {
    for at_b in [false, true] {
        for touch_end in [false, true] {
            let far_y = if touch_end { -3.0 } else { 3.0 };
            let mut sketch = shared_line_arc(at_b, touch_end, far_y, 1.0);
            sketch
                .add_line_tangent(constraint(1), entity(6), entity(4))
                .unwrap();
            constrain_shared_line(&mut sketch, 1.0);
            let original = sketch.clone();
            let result = sketch
                .solve(&parameters(), SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.rank, 8);
            assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
            assert_point(
                result.candidate(),
                5,
                [5.0, if touch_end { -1.0 } else { 1.0 }],
            );
            assert_eq!(result.candidate().constraints(), original.constraints());
            assert_eq!(sketch, original);
        }
    }
}
fn shared_arc_pair(second_x: f64, reversed: bool) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, p) in [
        (1, [0.0, 0.0]),
        (2, [2.0, 0.0]),
        (3, [0.0, 2.0]),
        (4, [second_x, 0.2]),
        (5, [second_x, 1.2]),
    ] {
        sketch.add_point(entity(id), p).unwrap();
    }
    sketch
        .add_arc(entity(6), entity(1), [entity(2), entity(3)], true)
        .unwrap();
    sketch
        .add_arc(entity(7), entity(4), [entity(2), entity(5)], false)
        .unwrap();
    let curves = if reversed {
        [entity(7), entity(6)]
    } else {
        [entity(6), entity(7)]
    };
    sketch.add_circular_tangent(constraint(1), curves).unwrap();
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Fix {
            point: entity(2),
            position: [2.0, 0.0],
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    relation(
        &mut sketch,
        5,
        PlanarConstraint::Vertical {
            points: [entity(4), entity(5)],
        },
    );
    relation(
        &mut sketch,
        6,
        PlanarConstraint::Distance {
            points: [entity(1), entity(4)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
    sketch
}
#[test]
fn shared_arc_contact_keeps_rank_at_exact_contact_and_drives_all_enclosure_orders() {
    for reversed in [false, true] {
        for x in [1.0, 3.0] {
            let sketch = shared_arc_pair(x, reversed);
            let mut parameters = parameters();
            parameters.set_expression(1, &format!("{x} cm")).unwrap();
            let result = sketch
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.rank, 10);
            assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
            assert_point(result.candidate(), 4, [x, 0.0]);
            assert_point(result.candidate(), 5, [x, 1.0]);
            assert_eq!(result.candidate().constraints(), sketch.constraints());
            assert_eq!(result.candidate().arcs(), sketch.arcs());
            let contact = &sketch.constraints()[0].relation;
            let expected = if x > 2.0 {
                CircleContact::External
            } else if reversed {
                CircleContact::SecondContainsFirst
            } else {
                CircleContact::FirstContainsSecond
            };
            assert!(
                matches!(contact, PlanarConstraint::CircularTangent { contact, .. } if *contact == expected)
            );
            let mut rank_only = Sketch2D::default();
            for point in result.candidate().points() {
                rank_only.add_point(point.id, point.position).unwrap();
            }
            for arc in sketch.arcs() {
                rank_only
                    .add_arc(
                        arc.id,
                        arc.center,
                        [arc.start, arc.end],
                        arc.counter_clockwise,
                    )
                    .unwrap();
            }
            let before = rank_only.analyze(&parameters, 1e-9).unwrap().rank;
            rank_only
                .add_circular_tangent(
                    constraint(1),
                    if reversed {
                        [entity(7), entity(6)]
                    } else {
                        [entity(6), entity(7)]
                    },
                )
                .unwrap();
            assert_eq!(before, 2);
            assert_eq!(
                rank_only.analyze(&parameters, 1e-9).unwrap().rank,
                before + 1
            );
        }
    }
}
#[test]
fn multiple_shared_contacts_are_named_declines_and_never_first_match_choices() {
    let mut sketch = shared_line_arc(false, false, 1.0, 1.0);
    sketch.add_line(entity(8), [entity(2), entity(3)]).unwrap();
    sketch
        .add_arc(entity(9), entity(5), [entity(3), entity(2)], false)
        .unwrap();
    let original = sketch.clone();
    for relation in [
        PlanarConstraint::LineTangent {
            line: entity(8),
            curve: entity(4),
            side: LineContactSide::Left,
        },
        PlanarConstraint::CircularTangent {
            curves: [entity(4), entity(9)],
            contact: CircleContact::External,
        },
    ] {
        assert!(
            sketch
                .add_constraint(constraint(99), relation)
                .unwrap_err()
                .to_string()
                .contains("at most one")
        );
    }
    assert!(
        sketch
            .add_line_tangent(constraint(99), entity(8), entity(4))
            .is_err()
    );
    assert!(
        sketch
            .add_circular_tangent(constraint(99), [entity(4), entity(9)])
            .is_err()
    );
    assert_eq!(sketch, original);
}
#[test]
fn shared_line_rank_and_solve_survive_large_and_small_working_units() {
    for scale in [1e-100, 1.0, 1e100] {
        let mut sketch = shared_line_arc(false, false, 3.0, scale);
        sketch
            .add_line_tangent(constraint(1), entity(6), entity(4))
            .unwrap();
        constrain_shared_line(&mut sketch, scale);
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert!(
            (result.candidate().point(entity(5)).unwrap().position[1] / scale - 1.0).abs() < 1e-8
        );
    }
}

#[test]
fn a_shared_contact_cannot_change_to_external_after_an_incompatible_dimension_edit() {
    let sketch = shared_arc_pair(1.0, false);
    let original = sketch.clone();
    let mut parameters = parameters();
    parameters.set_expression(1, "3 cm").unwrap();
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert!(result.numerical().rejected_trial.is_some());
    assert_eq!(result.candidate().constraints(), original.constraints());
    assert_eq!(sketch, original);
}

#[test]
fn equal_coordinate_points_with_distinct_ids_remain_free_contact() {
    let mut sketch = shared_line_arc(false, false, 1.0, 1.0);
    sketch.add_point(entity(9), [0.0, 1.0]).unwrap();
    sketch.add_line(entity(10), [entity(9), entity(5)]).unwrap();
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
            curve: entity(4),
            parameter: 1,
        },
    );
    for (id, point) in [(3, 2), (4, 3), (5, 9)] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Vertical {
                points: [entity(1), entity(point)],
            },
        );
    }
    relation(
        &mut sketch,
        6,
        PlanarConstraint::Horizontal {
            points: [entity(9), entity(5)],
        },
    );
    relation(
        &mut sketch,
        7,
        PlanarConstraint::Distance {
            points: [entity(9), entity(5)],
            measure: DistanceMeasure::Horizontal,
            parameter: 2,
        },
    );
    sketch
        .add_line_tangent(constraint(8), entity(10), entity(4))
        .unwrap();
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.rank, 10);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    assert_point(result.candidate(), 9, [0.0, 5.0]);
    assert_point(result.candidate(), 5, [2.5, 5.0]);
    assert_ne!(
        result.candidate().line(entity(10)).unwrap().endpoints[0],
        result.candidate().arc(entity(4)).unwrap().start
    );
}
