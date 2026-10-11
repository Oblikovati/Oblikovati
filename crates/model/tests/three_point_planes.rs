// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition, WorkPointReference},
        work_plane::{WorkPlaneRecipe, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

fn source() -> WorkGeometryDefinition {
    WorkGeometryDefinition::ThroughThreePoints([
        WorkPointReference::Origin,
        WorkPointReference::Feature(FeatureId(1)),
        WorkPointReference::Feature(FeatureId(2)),
    ])
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    append(
        &mut program,
        1,
        WorkGeometryDefinition::Point([1.0, 0.0, 0.0].map(DatumLength::Constant)),
    );
    append(
        &mut program,
        2,
        WorkGeometryDefinition::Point([
            DatumLength::Constant(0.0),
            DatumLength::Parameter(1),
            DatumLength::Constant(0.0),
        ]),
    );
    append(&mut program, 3, source());
    program
}
fn append(
    program: &mut FeatureSequence<WorkGeometryDefinition>,
    id: u64,
    definition: WorkGeometryDefinition,
) {
    program
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn parameters(value: &str) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "height", USER_PARAM, value)
        .unwrap();
    parameters
}
fn evaluated(program: &FeatureSequence<WorkGeometryDefinition>, value: &str) -> SketchPlane {
    program
        .evaluate_work_geometry(&parameters(value))
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
#[test]
fn current_point_drivers_and_authored_flip_determine_frame_orientation() {
    let mut program = program();
    assert_eq!(evaluated(&program, "2 cm").normal(), [0.0, 0.0, 1.0]);
    assert_eq!(evaluated(&program, "-2 cm").normal(), [0.0, 0.0, -1.0]);
    program.flip_plane_normal(FeatureId(3)).unwrap();
    assert_eq!(evaluated(&program, "-2 cm").normal(), [0.0, 0.0, 1.0]);
    assert_eq!(evaluated(&program, "-2 cm").recipe().origin, [0.0; 3]);
}
#[test]
fn driver_failure_has_no_stale_frame_and_recovery_uses_current_values() {
    let program = program();
    let failed = program.evaluate_work_geometry(&parameters("0 cm")).unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(failed.point(FeatureId(1)).is_ok());
    assert!(matches!(
        failed.health[2].status,
        oblikovati_model::feature::FeatureEvaluationStatus::Failed(_)
    ));
    assert_eq!(evaluated(&program, "4 cm").normal(), [0.0, 0.0, 1.0]);
}
#[test]
fn suppression_and_cutoff_do_not_expose_a_cached_plane() {
    let mut program = program();
    program.set_suppressed(FeatureId(2), true).unwrap();
    assert!(
        program
            .evaluate_work_geometry(&parameters("2 cm"))
            .unwrap()
            .plane(FeatureId(3))
            .is_err()
    );
    program.set_suppressed(FeatureId(2), false).unwrap();
    program.set_end_of_part(Some(2)).unwrap();
    assert!(
        program
            .evaluate_work_geometry(&parameters("2 cm"))
            .unwrap()
            .plane(FeatureId(3))
            .is_err()
    );
}
#[test]
fn source_reads_are_exact_and_invalid_dependencies_cannot_replace_the_recipe() {
    let mut program = program();
    let expected = [1, 2].map(|id| FeatureReadKey::Feature(FeatureId(id)));
    assert_eq!(
        source().reads().keys().iter().copied().collect::<Vec<_>>(),
        expected
    );
    let before = program.clone();
    for id in [3, 99] {
        let definition = WorkGeometryDefinition::ThroughThreePoints(
            [WorkPointReference::Feature(FeatureId(id)); 3],
        );
        assert!(
            program
                .replace(FeatureId(3), definition, definition.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&before));
    }
}
#[test]
fn wrong_kind_driver_is_a_named_local_decline() {
    let mut program = program();
    let definition = WorkGeometryDefinition::Fixed(SketchPlane::xy());
    program
        .replace(FeatureId(2), definition, definition.reads())
        .unwrap();
    let evaluation = program.evaluate_work_geometry(&parameters("2 cm")).unwrap();
    assert!(evaluation.plane(FeatureId(3)).is_err());
    let reason = format!("{:?}", evaluation.health[2]);
    assert!(reason.contains("point") && reason.contains("plane"));
}
#[test]
fn source_order_and_versioned_storage_survive_exact_round_trips() {
    let original = program();
    let recipe = original.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 11);
    assert_eq!(recipe.features[2].definition.version, 6);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(original.same_work_geometry_recipe(&restored));
    let mut legacy = recipe.clone();
    legacy.version = 10;
    assert!(FeatureSequence::from_work_geometry_recipe(&legacy).is_err());
    assert!(
        !source().same_recipe(WorkGeometryDefinition::ThroughThreePoints(
            [WorkPointReference::Origin; 3]
        ))
    );
}
#[test]
fn unknown_source_fields_and_source_versions_are_refused() {
    let mut recipe = source().recipe();
    recipe.version = 5;
    assert!(WorkGeometryDefinition::from_recipe(recipe).is_err());
    let mut value = serde_json::to_value(source().recipe()).unwrap();
    value["unknown"] = true.into();
    assert!(serde_json::from_value::<WorkPlaneRecipe>(value).is_err());
}
#[test]
fn generated_extreme_scales_keep_ordered_frames_without_cross_product_overflow() {
    for scale in [f64::from_bits(1), 1e-200, 1.0, 1e200, f64::MAX] {
        let frame = SketchPlane::through_points(
            [0.0; 3],
            [scale, 0.0, 0.0],
            [0.0, scale, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_xy_frame(frame);
        let reversed = SketchPlane::through_points(
            [0.0; 3],
            [0.0, scale, 0.0],
            [scale, 0.0, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_eq!(reversed.normal(), [0.0, 0.0, -1.0]);
    }
}
#[test]
fn repeated_and_collinear_points_and_nonfinite_inputs_have_no_frame() {
    for third in [[0.0; 3], [2.0, 0.0, 0.0], [f64::NAN, 1.0, 0.0]] {
        assert!(
            SketchPlane::through_points([0.0; 3], [1.0, 0.0, 0.0], third, Default::default())
                .is_err()
        );
    }
    assert!(
        SketchPlane::through_points([0.0; 3], [0.0; 3], [0.0, 1.0, 0.0], Default::default())
            .is_err()
    );
}
#[test]
fn first_point_signed_bits_and_overflowing_displacements_are_preserved() {
    let first = [-f64::MAX, -0.0, 0.0];
    let frame = SketchPlane::through_points(
        first,
        [f64::MAX, 0.0, 0.0],
        [-f64::MAX, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    assert_eq!(
        frame.recipe().origin.map(f64::to_bits),
        first.map(f64::to_bits)
    );
    assert_eq!(frame.normal(), [0.0, 0.0, 1.0]);
}

fn assert_xy_frame(frame: SketchPlane) {
    assert_eq!(frame.recipe().x_axis, [1.0, 0.0, 0.0]);
    assert_eq!(frame.normal(), [0.0, 0.0, 1.0]);
}

#[test]
fn oblique_frames_match_the_analytic_normal_and_contain_all_driver_points() {
    let points = [[1.0, 2.0, 3.0], [2.0, 3.0, 4.0], [0.0, 3.0, 3.0]];
    let frame =
        SketchPlane::through_points(points[0], points[1], points[2], Default::default()).unwrap();
    let expected = [-1.0, -1.0, 2.0].map(|value| value / 6.0_f64.sqrt());
    let bound = frame.options().angular_deviation;
    for (actual, wanted) in frame.normal().into_iter().zip(expected) {
        assert!((actual - wanted).abs() <= bound);
    }
    for point in points {
        let projected = frame.to_model(frame.to_sketch(point).unwrap()).unwrap();
        assert!(
            projected
                .into_iter()
                .zip(point)
                .all(|(actual, wanted)| (actual - wanted).abs() <= bound)
        );
    }
}

#[test]
fn generated_near_collinear_points_preserve_a_defined_normal() {
    for separation in [f64::from_bits(1), 1e-150, 1e-30, 1e-12] {
        let frame = SketchPlane::through_points(
            [0.0; 3],
            [1.0, 0.0, 0.0],
            [1.0, separation, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_xy_frame(frame);
    }
}
