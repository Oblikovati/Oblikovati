// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::LineContactSide;
fn dimensioned_point(side: LineContactSide, diameter: bool, scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [4.0, 0.0]), (3, [0.0, 0.0])] {
        sketch
            .add_point(entity(id), position.map(|v| v * scale))
            .unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    for id in [1, 2] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(id),
                position: if id == 1 {
                    [0.0, 0.0]
                } else {
                    [4.0 * scale, 0.0]
                },
            },
        );
    }
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Offset {
            point: entity(3),
            line: entity(4),
            parameter: 3,
            side,
            linear_diameter: diameter,
        },
    );
    sketch
}
fn offset_parameters(value: f64) -> ParameterCollection<u64> {
    let mut parameters = parameters();
    parameters
        .set_expression(1, &format!("{value} cm"))
        .unwrap();
    parameters
        .insert_expression(3, "point_offset", USER_PARAM, "width / 2")
        .unwrap();
    parameters
}
#[test]
fn offset_dimensions_keep_zero_dof_at_zero_and_after_formula_changes_for_both_sides() {
    for side in [LineContactSide::Left, LineContactSide::Right] {
        for diameter in [false, true] {
            let mut candidate = dimensioned_point(side, diameter, 1.0);
            let original = candidate.clone();
            let sign = if side == LineContactSide::Left {
                1.0
            } else {
                -1.0
            };
            for width in [0.0, 8.0, 2.0, 0.0, 4.0] {
                let result = candidate
                    .solve(&offset_parameters(width), SolveOptions::default(), || false)
                    .unwrap();
                assert_eq!(result.numerical().termination, SolveTermination::Converged);
                assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
                assert_eq!(result.numerical().structure.rank, 6);
                assert_point(
                    result.candidate(),
                    3,
                    [0.0, sign * width / if diameter { 4.0 } else { 2.0 }],
                );
                assert_eq!(result.parameter_dependencies(), [3]);
                assert_eq!(result.candidate().constraints(), original.constraints());
                assert_eq!(result.candidate().lines(), original.lines());
                candidate = result.into_candidate();
            }
        }
    }
}
#[test]
fn factory_selects_the_seed_side_without_moving_points_and_refuses_ambiguous_seeds() {
    for sign in [-1.0, 1.0] {
        let mut seed = dimensioned_point(LineContactSide::Left, false, 1.0);
        seed.set_position(entity(3), [0.0, sign * 2.0]).unwrap();
        let points = seed.points().to_vec();
        seed.add_offset_dimension(constraint(99), entity(3), entity(4), 3, true)
            .unwrap();
        assert_eq!(seed.points(), points);
        let PlanarConstraint::Offset {
            side,
            linear_diameter,
            ..
        } = seed.constraints().last().unwrap().relation
        else {
            panic!()
        };
        assert_eq!(
            side,
            if sign > 0.0 {
                LineContactSide::Left
            } else {
                LineContactSide::Right
            }
        );
        assert!(linear_diameter);
    }
    let mut sketch = dimensioned_point(LineContactSide::Left, false, 1.0);
    let original = sketch.clone();
    assert!(matches!(
        sketch.add_offset_dimension(constraint(99), entity(3), entity(4), 3, false),
        Err(SketchError::ContactSelection { .. })
    ));
    assert_eq!(sketch, original);
    for (point, line) in [(entity(999), entity(4)), (entity(3), entity(999))] {
        assert!(
            sketch
                .add_offset_dimension(constraint(99), point, line, 3, false)
                .unwrap_err()
                .to_string()
                .contains("999")
        );
    }
    assert_eq!(sketch, original);
    sketch.set_position(entity(2), [0.0, 0.0]).unwrap();
    assert!(matches!(
        sketch.add_offset_dimension(constraint(99), entity(3), entity(4), 3, false),
        Err(SketchError::ContactSelection { .. })
    ));
}
#[test]
fn invalid_targets_and_collapsed_line_errors_name_the_original_constraint() {
    let sketch = dimensioned_point(LineContactSide::Right, true, 1.0);
    let mut parameters = offset_parameters(4.0);
    for expression in ["-1 cm", "1 deg", "5e-324 cm", "sqrt(-1) * 1 cm"] {
        parameters.set_expression(3, expression).unwrap();
        assert!(
            sketch
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap_err()
                .to_string()
                .contains("parameter 3")
        );
    }
    parameters.set_expression(3, "2 cm").unwrap();
    let mut collapsed = sketch.clone();
    collapsed.set_position(entity(2), [0.0, 0.0]).unwrap();
    assert!(
        matches!(collapsed.solve(&parameters, SolveOptions::default(), || false), Err(SketchError::Residual { constraint: id, .. }) if id == constraint(4))
    );
    let mut repeated = Sketch2D::default();
    for point in sketch.points() {
        repeated.add_point(point.id, point.position).unwrap();
    }
    repeated
        .add_line(entity(4), [entity(1), entity(2)])
        .unwrap();
    relation(
        &mut repeated,
        1,
        PlanarConstraint::Offset {
            point: entity(1),
            line: entity(4),
            parameter: 3,
            side: LineContactSide::Left,
            linear_diameter: false,
        },
    );
    let result = repeated
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Stalled);
    assert_eq!(result.conflicting_constraints(), [constraint(1)]);
    assert_eq!(result.candidate(), &repeated);
}
#[test]
fn offset_solve_rank_cancellation_and_opposite_side_warm_starts_preserve_units() {
    for scale in [1e-100, 1.0, 1e100] {
        for diameter in [false, true] {
            let mut sketch = dimensioned_point(LineContactSide::Right, diameter, scale);
            sketch.set_position(entity(3), [0.0, 3.0 * scale]).unwrap();
            let parameters = offset_parameters(8.0 * scale);
            let result = sketch
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.rank, 6);
            let y = result.candidate().point(entity(3)).unwrap().position[1] / scale;
            assert!((y + if diameter { 2.0 } else { 4.0 }).abs() < 1e-8);
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
