// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn minor_driving() -> (Sketch2D, ParameterCollection<u64>) {
    let (mut sketch, mut parameters) = driving();
    parameters
        .insert_expression(2, "minor", USER_PARAM, "2 cm")
        .unwrap();
    let input = oblikovati_model::sketch::SketchDimensionalConstraintCreation::Measured(
        oblikovati_model::sketch::MeasuredDimensionOperands::EllipseMinorRadius {
            ellipse: SketchEntityId(2),
        },
    );
    input.append(&mut sketch, SketchConstraintId(2), 2).unwrap();
    (sketch, parameters)
}
#[test]
fn minor_radius_has_one_exact_derivative_and_keeps_major_radius_and_frame() {
    let (sketch, parameters) = minor_driving();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 3);
    assert_eq!(
        outcome.numerical().evaluation.jacobian()[1],
        vec![0.0, 0.0, 0.0, 1.0, 0.0]
    );
    assert_minor_measurement(outcome.candidate());
    assert_eq!(sketch.parameter_dependencies(), [1, 2]);
}
#[test]
fn independent_radii_center_and_axis_constraints_produce_zero_freedoms() {
    let (mut sketch, parameters) = minor_driving();
    let center = constrain_ellipse_frame(&mut sketch);
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(
        outcome.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(outcome.candidate().point(center.id).unwrap(), &center);
    let value = outcome.candidate().ellipse(SketchEntityId(2)).unwrap();
    assert!(value.major_axis()[1].abs() < 1e-8);
    assert!((value.radii()[0] - 4.0).abs() < 1e-8);
    assert!((value.radii()[1] - 2.0).abs() < 1e-8);
}
#[test]
fn reference_minor_radius_needs_no_healthy_target_and_keeps_geometry() {
    let (mut sketch, mut parameters) = minor_driving();
    sketch
        .set_dimension_reference(SketchConstraintId(2), true)
        .unwrap();
    parameters.set_expression(2, "missing").unwrap();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 4);
    assert_eq!(
        outcome
            .candidate()
            .measure_dimension(SketchConstraintId(2))
            .unwrap()
            .value,
        1.0
    );
    assert_eq!(sketch.parameter_dependencies(), [1]);
}
#[test]
fn minor_dimension_schema_nine_round_trips_and_refuses_earlier_records() {
    let (sketch, _) = minor_driving();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 9);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    assert!(
        sketch
            .same_recipe(&Sketch2D::from_recipe(&serde_json::from_slice(&bytes).unwrap()).unwrap())
    );
    for version in 1..=8 {
        let mut legacy = recipe.clone();
        legacy.version = version;
        assert!(
            Sketch2D::from_recipe(&legacy)
                .unwrap_err()
                .to_string()
                .contains("expected version 9")
        );
    }
}

fn assert_minor_measurement(candidate: &Sketch2D) {
    let value = candidate.ellipse(SketchEntityId(2)).unwrap();
    assert!((value.radii()[0] - 4.0).abs() < 1e-8);
    assert!((value.radii()[1] - 2.0).abs() < 1e-8);
    assert_eq!(value.major_axis(), [0.6, 0.8]);
    let measurement = candidate.measure_dimension(SketchConstraintId(2)).unwrap();
    assert!((measurement.value - 2.0).abs() < 1e-8);
}
fn constrain_ellipse_frame(sketch: &mut Sketch2D) -> oblikovati_model::sketch::SketchPoint2D {
    let center = *sketch.point(SketchEntityId(1)).unwrap();
    let ground = PlanarConstraint::Ground {
        points: vec![center],
    };
    sketch
        .add_constraint(SketchConstraintId(3), ground)
        .unwrap();
    let alignment = PlanarConstraint::EllipseAxisAlignment {
        ellipse: SketchEntityId(2),
        major_axis: true,
        vertical: false,
    };
    sketch
        .add_constraint(SketchConstraintId(4), alignment)
        .unwrap();
    center
}

#[test]
fn minor_radius_refuses_invalid_targets_without_changing_the_input_owner() {
    for expression in ["0 cm", "-1 cm", "5 cm", "90 deg"] {
        let (sketch, mut parameters) = minor_driving();
        let before = sketch.clone();
        parameters.set_expression(2, expression).unwrap();
        let cause = sketch
            .solve(&parameters, Default::default(), || false)
            .unwrap_err();
        assert!(
            cause.to_string().contains("expected"),
            "{expression}: {cause}"
        );
        assert!(sketch.same_recipe(&before));
    }
}

#[path = "minor/degenerate.rs"]
mod degenerate;
