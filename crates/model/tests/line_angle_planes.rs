// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::{
            DatumAngle, DatumLength, WorkGeometryDefinition, WorkGeometryError, WorkPlaneLineAngle,
        },
        work_plane::{WorkPlaneOrigin, WorkPlaneReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};
use std::{
    error::Error,
    f64::consts::{FRAC_PI_2, PI},
};
type AngleProgram = FeatureSequence<WorkGeometryDefinition>;
fn source(angle: DatumAngle) -> WorkGeometryDefinition {
    WorkGeometryDefinition::AtAngleAboutAxis(angle_source(angle))
}
fn angle_source(angle: DatumAngle) -> WorkPlaneLineAngle {
    WorkPlaneLineAngle {
        line: WorkAxisReference::Feature(FeatureId(1)),
        base: WorkPlaneReference::Feature(FeatureId(2)),
        angle,
    }
}
fn append(program: &mut AngleProgram, id: u64, definition: WorkGeometryDefinition) {
    program
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn program(angle: DatumAngle) -> AngleProgram {
    let mut program = AngleProgram::default();
    let axis = WorkAxisLine::new([2.0, 3.0, 4.0], [1.0, 0.0, 0.0]).unwrap();
    append(&mut program, 1, WorkGeometryDefinition::Axis(axis));
    append(
        &mut program,
        2,
        WorkGeometryDefinition::Fixed(SketchPlane::xy()),
    );
    append(&mut program, 3, source(angle));
    program
}
fn parameters(expression: &str) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "swing", USER_PARAM, expression)
        .unwrap();
    parameters
}
fn frame(program: &AngleProgram, expression: &str) -> SketchPlane {
    program
        .evaluate_work_geometry(&parameters(expression))
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
fn assert_normal(actual: SketchPlane, expected: [f64; 3]) {
    let bound = actual.options().angular_deviation;
    assert!(
        actual
            .normal()
            .into_iter()
            .zip(expected)
            .all(|(a, b)| (a - b).abs() <= bound)
    );
}
fn replace(program: &mut AngleProgram, id: u64, definition: WorkGeometryDefinition) {
    program
        .replace(FeatureId(id), definition.clone(), definition.reads())
        .unwrap();
}
#[test]
fn current_angular_parameters_resolve_degrees_to_radians_without_length_scaling() {
    let program = program(DatumAngle::parameter(1));
    assert_normal(frame(&program, "90 deg"), [0.0, -1.0, 0.0]);
    assert_normal(frame(&program, "180 deg"), [0.0, 0.0, -1.0]);
    assert_normal(frame(&program, "-90 deg"), [0.0, 1.0, 0.0]);
    assert_eq!(frame(&program, "0 rad").recipe().origin, [2.0, 3.0, 4.0]);
    assert_eq!(
        DatumAngle::parameter(1)
            .resolve(&parameters("90 deg"))
            .unwrap(),
        FRAC_PI_2
    );
}
#[test]
fn current_axis_and_base_orientation_drive_rotation_but_base_origin_does_not() {
    let mut program = program(DatumAngle::constant(0.0).unwrap());
    let before = frame(&program, "0 rad");
    let mut recipe = SketchPlane::xy().recipe();
    recipe.origin = [f64::MAX; 3];
    let displaced = SketchPlane::from_recipe(recipe, Default::default()).unwrap();
    replace(&mut program, 2, WorkGeometryDefinition::Fixed(displaced));
    assert!(before.same_recipe(frame(&program, "0 rad")));
    let axis = WorkAxisLine::new([5.0, 6.0, 7.0], [0.0, 1.0, 0.0]).unwrap();
    replace(&mut program, 1, WorkGeometryDefinition::Axis(axis));
    replace(
        &mut program,
        2,
        WorkGeometryDefinition::Fixed(SketchPlane::yz()),
    );
    assert_eq!(frame(&program, "0 rad").recipe().origin, [5.0, 6.0, 7.0]);
    assert_normal(frame(&program, "0 rad"), [1.0, 0.0, 0.0]);
}
#[test]
fn authored_normal_flip_remains_separate_from_the_live_angle() {
    let mut program = program(DatumAngle::parameter(1));
    program.flip_plane_normal(FeatureId(3)).unwrap();
    assert_normal(frame(&program, "90 deg"), [0.0, 1.0, 0.0]);
    assert_normal(frame(&program, "0 rad"), [0.0, 0.0, -1.0]);
    program.flip_plane_normal(FeatureId(2)).unwrap();
    assert_normal(frame(&program, "0 rad"), [0.0, 0.0, 1.0]);
}
#[test]
fn missing_and_wrong_dimension_angles_fail_locally_and_recover() {
    let program = program(DatumAngle::parameter(1));
    let result = program.evaluate_work_geometry(&parameters("2 cm")).unwrap();
    assert!(result.plane(FeatureId(3)).is_err());
    assert!(format!("{:?}", result.health[2]).contains("Angle"));
    assert!(result.axis(FeatureId(1)).is_ok());
    let missing = ParameterCollection::new(Default::default());
    assert!(
        program
            .evaluate_work_geometry(&missing)
            .unwrap()
            .plane(FeatureId(3))
            .is_err()
    );
    assert_normal(frame(&program, "90 deg"), [0.0, -1.0, 0.0]);
}
#[test]
fn perpendicular_base_and_suppressed_axis_have_no_current_frame() {
    let mut program = program(DatumAngle::constant(0.0).unwrap());
    replace(
        &mut program,
        2,
        WorkGeometryDefinition::Fixed(SketchPlane::yz()),
    );
    assert_decline(&program);
    replace(
        &mut program,
        2,
        WorkGeometryDefinition::Fixed(SketchPlane::xy()),
    );
    program.set_suppressed(FeatureId(1), true).unwrap();
    assert_decline(&program);
    program.set_suppressed(FeatureId(1), false).unwrap();
    assert_normal(frame(&program, "0 rad"), [0.0, 0.0, 1.0]);
}
fn assert_decline(program: &AngleProgram) {
    let evaluated = program
        .evaluate_work_geometry(&parameters("0 rad"))
        .unwrap();
    assert!(evaluated.plane(FeatureId(3)).is_err());
}
#[test]
fn reads_include_both_geometry_drivers_and_the_angular_parameter() {
    let definition = source(DatumAngle::parameter(8));
    let expected = [
        FeatureReadKey::Feature(FeatureId(1)),
        FeatureReadKey::Feature(FeatureId(2)),
        FeatureReadKey::Parameter(8),
    ];
    let reads = definition.reads();
    assert_eq!(reads.keys().iter().copied().collect::<Vec<_>>(), expected);
    let origin = WorkGeometryDefinition::AtAngleAboutAxis(WorkPlaneLineAngle {
        line: WorkAxisReference::OriginX,
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        angle: DatumAngle::constant(0.0).unwrap(),
    });
    assert!(origin.reads().keys().is_empty());
}
#[test]
fn self_forward_and_wrong_kind_drivers_are_refused() {
    let mut program = program(DatumAngle::constant(0.0).unwrap());
    let before = program.clone();
    for id in [3, 99] {
        let mut angle = angle_source(DatumAngle::parameter(1));
        angle.base = WorkPlaneReference::Feature(FeatureId(id));
        let invalid = WorkGeometryDefinition::AtAngleAboutAxis(angle);
        assert!(
            program
                .replace(FeatureId(3), invalid.clone(), invalid.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&before));
    }
    let wrong_kind = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    replace(&mut program, 2, WorkGeometryDefinition::Axis(wrong_kind));
    assert_decline(&program);
}
#[test]
fn source_wrappers_and_program_versions_round_trip_with_orientation() {
    let mut program = program(DatumAngle::parameter(1));
    program.flip_plane_normal(FeatureId(3)).unwrap();
    let recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(
        (recipe.version, recipe.features[2].definition.version),
        (15, 10)
    );
    let restored = AngleProgram::from_work_geometry_recipe(&recipe).unwrap();
    assert!(program.same_work_geometry_recipe(&restored));
    assert!(frame(&program, "45 deg").same_recipe(frame(&restored, "45 deg")));
    let mut old = recipe.clone();
    old.version = 14;
    assert!(AngleProgram::from_work_geometry_recipe(&old).is_err());
    let mut definition = recipe.features[2].definition.clone();
    definition.version = 9;
    assert!(WorkGeometryDefinition::from_recipe(definition.clone()).is_err());
}
#[test]
fn scalar_source_equality_preserves_signed_zero_and_parameter_identity() {
    let positive = DatumAngle::constant(0.0).unwrap();
    let negative = DatumAngle::constant(-0.0).unwrap();
    assert!(!positive.same_recipe(negative));
    assert!(!source(positive).same_recipe(&source(negative)));
    assert!(source(DatumAngle::parameter(1)).same_recipe(&source(DatumAngle::parameter(1))));
    assert!(!source(DatumAngle::parameter(1)).same_recipe(&source(DatumAngle::parameter(2))));
    assert_eq!(positive.parameter_id(), None);
    assert_eq!(DatumAngle::parameter(3).parameter_id(), Some(3));
    assert_eq!(
        negative.resolve(&parameters("0 rad")).unwrap().to_bits(),
        (-0.0_f64).to_bits()
    );
}
#[test]
fn angular_declines_retain_offending_quantities_and_parameter_causes() {
    for invalid in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let error = DatumAngle::constant(invalid).unwrap_err();
        assert!(error.to_string().contains("finite Angle in radians"));
        assert!(error.source().is_some());
    }
}
#[test]
fn missing_and_dimension_mismatched_scalar_sources_retain_their_causes() {
    let missing = DatumAngle::parameter(99)
        .resolve(&parameters("1 rad"))
        .unwrap_err();
    assert!(matches!(missing, WorkGeometryError::Parameter(_)));
    assert!(missing.source().is_some());
    let wrong = DatumAngle::parameter(1)
        .resolve(&parameters("1 cm"))
        .unwrap_err();
    assert!(wrong.to_string().contains("Some(1)"));
    assert!(wrong.to_string().contains("Length"));
    assert!(
        DatumLength::Parameter(1)
            .resolve(&parameters("1 rad"))
            .is_err()
    );
}
#[test]
fn unknown_constructor_fields_do_not_restore() {
    let mut value = serde_json::to_value(source(DatumAngle::parameter(1)).recipe()).unwrap();
    value["definition"]["AtAngleAboutAxis"]["unknown"] = true.into();
    assert!(
        serde_json::from_value::<oblikovati_model::feature::work_plane::WorkPlaneRecipe>(value)
            .is_err()
    );
}
#[test]
fn principal_rotations_match_analytic_right_hand_results() {
    for (radians, expected) in [
        (0.0, [0.0, 0.0, 1.0]),
        (FRAC_PI_2, [0.0, -1.0, 0.0]),
        (-FRAC_PI_2, [0.0, 1.0, 0.0]),
        (PI, [0.0, 0.0, -1.0]),
    ] {
        let actual = SketchPlane::xy()
            .at_angle_about_axis([2.0, 3.0, 4.0], [1.0, 0.0, 0.0], radians)
            .unwrap();
        assert_normal(actual, expected);
        assert_eq!(actual.recipe().origin, [2.0, 3.0, 4.0]);
    }
}
#[test]
fn generated_oblique_axes_keep_the_line_contained_and_the_normal_repeatable() {
    for index in -24..=24 {
        let axis = WorkAxisLine::new([f64::MAX, -0.0, 0.0], [f64::from(index), 2.0, 3.0]).unwrap();
        for angle in [-PI, -FRAC_PI_2, 0.0, FRAC_PI_2, PI] {
            let actual = rotated_axis(axis, angle);
            let repeat = rotated_axis(axis, angle);
            assert!(actual.same_recipe(repeat));
            assert_contains_axis(actual, axis);
            assert_rotation_angle(actual, axis, angle);
            assert_eq!(
                actual.recipe().origin.map(f64::to_bits),
                axis.origin().map(f64::to_bits)
            );
        }
    }
}
#[test]
fn invalid_rotation_inputs_and_parallel_oblique_normals_are_named_declines() {
    let base = SketchPlane::xy();
    for invalid in [f64::NAN, f64::INFINITY] {
        assert!(
            base.at_angle_about_axis([0.0; 3], [1.0, 0.0, 0.0], invalid)
                .is_err()
        );
    }
    assert!(
        base.at_angle_about_axis([f64::INFINITY; 3], [1.0, 0.0, 0.0], 0.0)
            .is_err()
    );
    assert!(
        base.at_angle_about_axis([0.0; 3], [2.0, 0.0, 0.0], 0.0)
            .is_err()
    );
}
#[test]
fn parallel_oblique_base_normals_refuse_undefined_reference_roll() {
    let axis = WorkAxisLine::new([0.0; 3], [2.0, 3.0, 4.0]).unwrap();
    let oblique =
        SketchPlane::from_origin_normal([0.0; 3], axis.direction(), Default::default()).unwrap();
    assert!(
        oblique
            .at_angle_about_axis([0.0; 3], axis.direction(), FRAC_PI_2)
            .is_err()
    );
}

fn rotated_axis(axis: WorkAxisLine, radians: f64) -> SketchPlane {
    SketchPlane::xy()
        .at_angle_about_axis(axis.origin(), axis.direction(), radians)
        .unwrap()
}

fn assert_contains_axis(frame: SketchPlane, axis: WorkAxisLine) {
    let dot: f64 = frame
        .normal()
        .into_iter()
        .zip(axis.direction())
        .map(|(a, b)| a * b)
        .sum();
    assert!(dot.abs() <= frame.options().angular_deviation);
}

fn assert_rotation_angle(frame: SketchPlane, axis: WorkAxisLine, radians: f64) {
    let reference = rotated_axis(axis, 0.0).normal();
    let actual = frame.normal();
    let cosine: f64 = reference.into_iter().zip(actual).map(|(a, b)| a * b).sum();
    let cross = [
        reference[1] * actual[2] - reference[2] * actual[1],
        reference[2] * actual[0] - reference[0] * actual[2],
        reference[0] * actual[1] - reference[1] * actual[0],
    ];
    let sine: f64 = cross
        .into_iter()
        .zip(axis.direction())
        .map(|(a, b)| a * b)
        .sum();
    let bound = frame.options().angular_deviation;
    assert!((cosine - radians.cos()).abs() <= bound);
    assert!((sine - radians.sin()).abs() <= bound);
}
