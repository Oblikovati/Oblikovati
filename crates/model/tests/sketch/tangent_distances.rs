// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    LineContactSide, TangentDistanceContact, TangentDistanceOptions, TangentLean,
};
fn offset_parameters(value: f64) -> ParameterCollection<u64> {
    let mut parameters = parameters();
    parameters
        .insert_expression(3, "offset", USER_PARAM, &format!("{value} cm"))
        .unwrap();
    parameters
}
fn offset_sketch(side: f64, options: TangentDistanceOptions) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, xy) in [(1, [0.0, 0.0]), (2, [4.0, 0.0]), (3, [0.0, side * 5.0])] {
        sketch.add_point(entity(id), xy).unwrap();
    }
    sketch.add_line(entity(5), [entity(1), entity(2)]).unwrap();
    sketch.add_circle(entity(6), entity(3), 5.0).unwrap();
    for id in [1, 2] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(id),
                position: sketch_point(id),
            },
        );
    }
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Radius {
            curve: entity(6),
            parameter: 1,
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    sketch
        .add_tangent_distance(constraint(6), entity(5), entity(6), 3, options)
        .unwrap();
    sketch
}
fn sketch_point(id: u64) -> [f64; 2] {
    if id == 1 { [0.0, 0.0] } else { [4.0, 0.0] }
}
#[test]
fn free_circle_offsets_are_signed_and_doubled_without_changing_the_picked_side() {
    for side in [-1.0, 1.0] {
        for far_side in [false, true] {
            for linear_diameter in [false, true] {
                let options = TangentDistanceOptions {
                    far_side,
                    linear_diameter,
                };
                let mut sketch = offset_sketch(side, options);
                for distance in [3.0, 7.0, 5.0] {
                    let offset = if far_side {
                        distance + 5.0
                    } else {
                        distance - 5.0
                    };
                    let parameters =
                        offset_parameters(offset * if linear_diameter { 2.0 } else { 1.0 });
                    let result = sketch
                        .solve(&parameters, SolveOptions::default(), || false)
                        .unwrap();
                    assert_eq!(result.numerical().termination, SolveTermination::Converged);
                    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
                    assert_point(result.candidate(), 3, [0.0, side * distance]);
                    assert_eq!(result.parameter_dependencies(), [1, 3]);
                    assert_eq!(result.candidate().constraints(), sketch.constraints());
                    sketch = result.into_candidate();
                }
            }
        }
    }
}
fn shared_sketch(side: f64, options: TangentDistanceOptions) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, xy) in [
        (1, [0.0, 0.0]),
        (2, [4.0, 0.0]),
        (3, [0.0, side * 5.0]),
        (4, [0.0, side * 10.0]),
    ] {
        sketch.add_point(entity(id), xy).unwrap();
    }
    sketch.add_line(entity(5), [entity(1), entity(2)]).unwrap();
    sketch
        .add_arc(entity(6), entity(3), [entity(1), entity(4)], side > 0.0)
        .unwrap();
    for id in [1, 2] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(id),
                position: sketch_point(id),
            },
        );
    }
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Radius {
            curve: entity(6),
            parameter: 1,
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Vertical {
            points: [entity(3), entity(4)],
        },
    );
    sketch
        .add_tangent_distance(constraint(5), entity(5), entity(6), 3, options)
        .unwrap();
    sketch
}
#[test]
fn shared_contact_radius_elimination_has_full_rank_at_tangency_and_after_parameter_edits() {
    for side in [-1.0, 1.0] {
        for far_side in [false, true] {
            for linear_diameter in [false, true] {
                let options = TangentDistanceOptions {
                    far_side,
                    linear_diameter,
                };
                let original = shared_sketch(side, options);
                let mut candidate = original.clone();
                for distance in [5.0, 3.0, 4.0, 5.0] {
                    let offset = if far_side {
                        distance + 5.0
                    } else {
                        distance - 5.0
                    };
                    let parameters =
                        offset_parameters(offset * if linear_diameter { 2.0 } else { 1.0 });
                    let result = candidate
                        .solve(&parameters, SolveOptions::default(), || false)
                        .unwrap();
                    assert_eq!(
                        result.numerical().termination,
                        SolveTermination::Converged,
                        "distance {distance}: {:?}",
                        result.numerical()
                    );
                    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
                    assert_point(
                        result.candidate(),
                        3,
                        [(25.0 - distance * distance).sqrt(), side * distance],
                    );
                    assert_eq!(result.candidate().arcs(), original.arcs());
                    assert_eq!(result.candidate().constraints(), original.constraints());
                    candidate = result.into_candidate();
                }
            }
        }
    }
}
#[test]
fn invalid_targets_contact_identity_and_seed_side_are_atomic_named_errors() {
    for far_side in [false, true] {
        let options = TangentDistanceOptions {
            far_side,
            linear_diameter: true,
        };
        let sketch = shared_sketch(1.0, options);
        for expression in [
            "1 deg",
            "5e-324 cm",
            if far_side { "0 cm" } else { "1 cm" },
            if far_side { "-1 cm" } else { "-11 cm" },
        ] {
            let mut parameters = offset_parameters(1.0);
            parameters.set_expression(3, expression).unwrap();
            assert!(
                sketch
                    .solve(&parameters, SolveOptions::default(), || false)
                    .is_err(),
                "{expression}"
            );
        }
    }
    let mut sketch = shared_sketch(1.0, Default::default());
    let original = sketch.clone();
    for curve in [entity(6), entity(999)] {
        assert!(
            sketch
                .add_constraint(
                    constraint(99),
                    PlanarConstraint::TangentDistance {
                        line: entity(5),
                        curve,
                        parameter: 3,
                        options: Default::default(),
                        contact: TangentDistanceContact {
                            side: LineContactSide::Left,
                            lean: None
                        },
                    }
                )
                .is_err()
        );
    }
    assert_eq!(sketch, original);
    sketch.set_position(entity(3), [5.0, 0.0]).unwrap();
    assert!(matches!(
        sketch.add_tangent_distance(constraint(99), entity(5), entity(6), 3, Default::default()),
        Err(SketchError::ContactSelection { .. })
    ));
    assert_eq!(sketch.constraints(), original.constraints());
}
#[test]
fn negative_lean_diameter_driver_and_cancellation_preserve_the_authored_branch() {
    let mut sketch = shared_sketch(-1.0, Default::default());
    sketch.set_position(entity(3), [-3.0, -4.0]).unwrap();
    let mut rebuilt = Sketch2D::default();
    for point in sketch.points() {
        rebuilt.add_point(point.id, point.position).unwrap();
    }
    rebuilt.add_line(entity(5), [entity(2), entity(1)]).unwrap();
    rebuilt
        .add_arc(entity(6), entity(3), [entity(1), entity(4)], false)
        .unwrap();
    for constraint in &sketch.constraints()[..4] {
        let relation = if constraint.id == super::constraint(3) {
            PlanarConstraint::Diameter {
                curve: entity(6),
                parameter: 1,
            }
        } else {
            constraint.relation.clone()
        };
        rebuilt.add_constraint(constraint.id, relation).unwrap();
    }
    rebuilt
        .add_tangent_distance(constraint(5), entity(5), entity(6), 3, Default::default())
        .unwrap();
    let PlanarConstraint::TangentDistance { contact, .. } = rebuilt.constraints()[4].relation
    else {
        panic!()
    };
    assert_eq!(contact.lean, Some(TangentLean::Positive)); // reversed line retains its own orientation.
    let mut parameters = offset_parameters(-1.0);
    parameters.set_expression(1, "10 cm").unwrap();
    let cancelled = rebuilt
        .solve(&parameters, SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(
        cancelled.numerical().termination,
        SolveTermination::Cancelled
    );
    assert_eq!(cancelled.candidate(), &rebuilt);
    let result = rebuilt
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_point(result.candidate(), 3, [-3.0, -4.0]);
}
#[test]
fn free_arc_offsets_and_negative_shared_lean_solve_at_small_and_large_scales() {
    for scale in [1e-100, 1.0, 1e100] {
        let mut sketch = offset_sketch(1.0, Default::default());
        let mut arc = Sketch2D::default();
        for point in sketch.points() {
            arc.add_point(point.id, point.position.map(|v| v * scale))
                .unwrap();
        }
        arc.add_point(entity(7), [5.0 * scale, 5.0 * scale])
            .unwrap();
        arc.add_point(entity(8), [0.0, 10.0 * scale]).unwrap();
        arc.add_line(entity(5), [entity(1), entity(2)]).unwrap();
        arc.add_arc(entity(6), entity(3), [entity(7), entity(8)], true)
            .unwrap();
        for constraint in sketch.constraints() {
            let mut relation = constraint.relation.clone();
            if let PlanarConstraint::Fix {
                ref mut position, ..
            } = relation
            {
                *position = position.map(|v| v * scale);
            }
            arc.add_constraint(constraint.id, relation).unwrap();
        }
        relation(
            &mut arc,
            7,
            PlanarConstraint::Horizontal {
                points: [entity(3), entity(7)],
            },
        );
        relation(
            &mut arc,
            8,
            PlanarConstraint::Vertical {
                points: [entity(3), entity(8)],
            },
        );
        let mut parameters = offset_parameters(-2.0 * scale);
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let result = arc
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert!(
            (result.candidate().point(entity(3)).unwrap().position[1] / scale - 3.0).abs() < 1e-8
        );
        sketch = shared_sketch(1.0, Default::default());
        sketch.set_position(entity(3), [-3.0, 4.0]).unwrap();
        // Re-create the dimension to retain the picked negative lean, without changing geometry.
        let mut negative = Sketch2D::default();
        for point in sketch.points() {
            negative
                .add_point(point.id, point.position.map(|v| v * scale))
                .unwrap();
        }
        negative
            .add_line(entity(5), [entity(1), entity(2)])
            .unwrap();
        negative
            .add_arc(entity(6), entity(3), [entity(1), entity(4)], true)
            .unwrap();
        for constraint in &sketch.constraints()[..4] {
            let mut relation = constraint.relation.clone();
            if let PlanarConstraint::Fix {
                ref mut position, ..
            } = relation
            {
                *position = position.map(|v| v * scale);
            }
            negative.add_constraint(constraint.id, relation).unwrap();
        }
        negative
            .add_tangent_distance(constraint(5), entity(5), entity(6), 3, Default::default())
            .unwrap();
        let result = negative
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert!(
            (result.candidate().point(entity(3)).unwrap().position[0] / scale + 4.0).abs() < 1e-8
        );
    }
}
#[test]
fn shared_dimensions_keep_all_radius_sources_and_report_the_unfixed_singular_seed() {
    let mut seed = shared_sketch(1.0, Default::default());
    let mut conflicting = seed.clone();
    relation(
        &mut conflicting,
        99,
        PlanarConstraint::Radius {
            curve: entity(6),
            parameter: 2,
        },
    );
    let result = conflicting
        .solve(&offset_parameters(0.0), SolveOptions::default(), || false)
        .unwrap();
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert!(!result.conflicting_constraints().is_empty());
    seed.set_position(entity(4), [0.0, 10.0]).unwrap();
    let mut no_driver = Sketch2D::default();
    for point in seed.points() {
        no_driver.add_point(point.id, point.position).unwrap();
    }
    no_driver
        .add_line(entity(5), [entity(1), entity(2)])
        .unwrap();
    no_driver
        .add_arc(entity(6), entity(3), [entity(1), entity(4)], true)
        .unwrap();
    for constraint in seed
        .constraints()
        .iter()
        .filter(|c| c.id != super::constraint(3))
    {
        no_driver
            .add_constraint(constraint.id, constraint.relation.clone())
            .unwrap();
    }
    let tangent = no_driver
        .solve(&offset_parameters(0.0), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(tangent.numerical().termination, SolveTermination::Converged);
    assert_eq!(tangent.numerical().structure.degrees_of_freedom, 1);
    let result = no_driver
        .solve(&offset_parameters(-1.0), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Stalled);
    assert_eq!(result.candidate(), &no_driver);
    assert!(!result.conflicting_constraints().is_empty());
}
