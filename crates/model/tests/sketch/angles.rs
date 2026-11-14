// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::AngleTurn;
fn angled_rays(clockwise: bool, three_point: bool, scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, p) in [
        (1, [0.0, 0.0]),
        (2, [4.0 * scale, 0.0]),
        (
            3,
            [
                3.0 * scale,
                if clockwise { -2.0 * scale } else { 2.0 * scale },
            ],
        ),
    ] {
        sketch.add_point(entity(id), p).unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(5), [entity(1), entity(3)]).unwrap();
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
    for (id, point, parameter) in [(3, 2, 1), (4, 3, 2)] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Distance {
                points: [entity(1), entity(point)],
                measure: DistanceMeasure::Aligned,
                parameter,
            },
        );
    }
    if three_point {
        sketch
            .add_three_point_angle(constraint(5), entity(1), [entity(2), entity(3)], 3)
            .unwrap();
    } else {
        sketch
            .add_angle(constraint(5), [entity(4), entity(5)], 3)
            .unwrap();
    }
    sketch
}
fn angle_parameters(scale: f64) -> ParameterCollection<u64> {
    let mut parameters = parameters();
    parameters
        .set_expression(1, &format!("{} cm", 5.0 * scale))
        .unwrap();
    parameters
        .insert_expression(3, "bearing", USER_PARAM, "30 deg")
        .unwrap();
    parameters
}
#[test]
fn both_angle_forms_retain_their_turn_and_solve_edits_including_boundary_angles_with_zero_dof() {
    for clockwise in [false, true] {
        for three_point in [false, true] {
            let original = angled_rays(clockwise, three_point, 1.0);
            let mut candidate = original.clone();
            let mut parameters = angle_parameters(1.0);
            for degrees in [30.0_f64, 120.0, 0.0, 180.0] {
                parameters
                    .set_expression(3, &format!("{degrees} deg"))
                    .unwrap();
                let result = candidate
                    .solve(&parameters, SolveOptions::default(), || false)
                    .unwrap();
                assert_eq!(
                    result.numerical().termination,
                    SolveTermination::Converged,
                    "{clockwise} {three_point} {degrees}"
                );
                assert_eq!(result.numerical().structure.variables, 6);
                assert_eq!(result.numerical().structure.rank, 6);
                assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
                let angle = degrees.to_radians() * if clockwise { -1.0 } else { 1.0 };
                assert_point(
                    result.candidate(),
                    3,
                    [2.5 * angle.cos(), 2.5 * angle.sin()],
                );
                assert_eq!(result.candidate().constraints(), original.constraints());
                assert_eq!(result.parameter_dependencies(), [1, 2, 3]);
                candidate = result.into_candidate();
            }
        }
    }
}
#[test]
fn angular_model_values_and_rank_survive_coordinate_units_and_cancellation() {
    for scale in [1e-100, 1.0, 1e100] {
        let sketch = angled_rays(false, true, scale);
        let parameters = angle_parameters(scale);
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.rank, 6);
        let p = result.candidate().point(entity(3)).unwrap().position;
        assert!((p[0] / scale - 2.5 * 30.0_f64.to_radians().cos()).abs() < 1e-8);
        assert!((p[1] / scale - 1.25).abs() < 1e-8);
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
#[test]
fn invalid_angular_targets_keep_their_parameter_identity_and_expected_domain() {
    let sketch = angled_rays(false, true, 1.0);
    let mut parameters = angle_parameters(1.0);
    for expression in ["1 cm", "-1 deg", "181 deg"] {
        parameters.set_expression(3, expression).unwrap();
        let error = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap_err();
        assert!(error.to_string().contains("angle parameter 3"));
        assert!(error.to_string().contains("[0, pi]"));
    }
    parameters.set_expression(3, "1 / 0").unwrap();
    let error = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap_err();
    assert!(
        matches!(error, SketchError::Parameter { constraint: id, parameter: 3, .. } if id == constraint(5))
    );
    assert!(std::error::Error::source(&error).is_some());
}
#[test]
fn angle_creation_is_atomic_and_collapsed_sources_name_the_authored_relation() {
    let mut sketch = angled_rays(false, false, 1.0);
    let original = sketch.clone();
    assert!(
        sketch
            .add_angle(constraint(99), [entity(4), entity(999)], 3)
            .is_err()
    );
    assert!(
        sketch
            .add_three_point_angle(constraint(99), entity(999), [entity(2), entity(3)], 3)
            .is_err()
    );
    assert!(
        sketch
            .add_angle(constraint(5), [entity(4), entity(5)], 3)
            .is_err()
    );
    assert_eq!(sketch, original);
    sketch.set_position(entity(3), [0.0, 0.0]).unwrap();
    let error = sketch
        .add_angle(constraint(99), [entity(4), entity(5)], 3)
        .unwrap_err();
    assert!(
        matches!(error, SketchError::AngleSelection { constraint: id, .. } if id == constraint(99))
    );
    assert!(error.to_string().contains("angle selection"));
    assert!(std::error::Error::source(&error).is_some());
    let error = sketch
        .solve(&angle_parameters(1.0), SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(error, SketchError::Residual { constraint: id, .. } if id == constraint(5)));
}
#[test]
fn collinear_seed_ties_choose_counterclockwise_and_explicit_turn_can_override_the_seed() {
    let mut sketch = Sketch2D::default();
    for (id, p) in [(1, [0.0, 0.0]), (2, [1.0, 0.0]), (3, [-1.0, -0.0])] {
        sketch.add_point(entity(id), p).unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(5), [entity(1), entity(3)]).unwrap();
    sketch
        .add_angle(constraint(1), [entity(4), entity(5)], 3)
        .unwrap();
    assert!(matches!(
        sketch.constraints()[0].relation,
        PlanarConstraint::Angle {
            turn: AngleTurn::CounterClockwise,
            ..
        }
    ));
    let mut sketch = angled_rays(false, true, 1.0);
    sketch.set_position(entity(3), [3.0, -2.0]).unwrap();
    let result = sketch
        .solve(&angle_parameters(1.0), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert!(result.candidate().point(entity(3)).unwrap().position[1] > 0.0);
}
