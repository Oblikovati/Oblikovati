// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::{
            DatumLength, WorkGeometryDefinition, WorkPlaneLinePoint, WorkPointReference,
        },
        work_plane::WorkPlaneResolver,
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

fn source(line: WorkAxisReference, point: WorkPointReference) -> WorkGeometryDefinition {
    WorkGeometryDefinition::LineThroughPoint(WorkPlaneLinePoint { line, point })
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    let axis = WorkAxisLine::new([2.0, 3.0, 4.0], [1.0, 0.0, 0.0]).unwrap();
    append(&mut program, 1, WorkGeometryDefinition::Axis(axis));
    let point = [
        DatumLength::Constant(2.0),
        DatumLength::Parameter(1),
        DatumLength::Constant(4.0),
    ];
    append(&mut program, 2, WorkGeometryDefinition::Point(point));
    append(
        &mut program,
        3,
        source(
            WorkAxisReference::Feature(FeatureId(1)),
            WorkPointReference::Feature(FeatureId(2)),
        ),
    );
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
fn frame(program: &FeatureSequence<WorkGeometryDefinition>, value: &str) -> SketchPlane {
    program
        .evaluate_work_geometry(&parameters(value))
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
#[test]
fn current_axis_origin_direction_and_point_parameters_drive_the_frame() {
    let mut program = program();
    let initial = frame(&program, "5 cm");
    assert_eq!(initial.recipe().origin, [2.0, 3.0, 4.0]);
    assert_eq!(initial.normal(), [0.0, 0.0, 1.0]);
    assert_eq!(frame(&program, "1 cm").normal(), [0.0, 0.0, -1.0]);
    let axis = WorkAxisLine::new([0.0; 3], [0.0, 0.0, 1.0]).unwrap();
    let replacement = WorkGeometryDefinition::Axis(axis);
    program
        .replace(FeatureId(1), replacement, replacement.reads())
        .unwrap();
    assert_eq!(frame(&program, "0 cm").normal(), [0.0, 1.0, 0.0]);
    assert_eq!(frame(&program, "0 cm").recipe().origin, [0.0; 3]);
}
#[test]
fn all_origin_axes_resolve_without_feature_reads() {
    let current = FeatureSequence::default()
        .evaluate_work_geometry(&parameters("1 cm"))
        .unwrap();
    for (reference, direction) in [
        (WorkAxisReference::OriginX, [1.0, 0.0, 0.0]),
        (WorkAxisReference::OriginY, [0.0, 1.0, 0.0]),
        (WorkAxisReference::OriginZ, [0.0, 0.0, 1.0]),
    ] {
        let axis = reference.line(&current).unwrap();
        assert_eq!(axis.origin(), [0.0; 3]);
        assert_eq!(axis.direction(), direction);
        assert!(
            source(reference, WorkPointReference::Origin)
                .reads()
                .keys()
                .is_empty()
        );
    }
}
#[test]
fn collinearity_has_no_current_frame_and_valid_drivers_recover_with_saved_flip() {
    let mut program = program();
    program.flip_plane_normal(FeatureId(3)).unwrap();
    let failed = program.evaluate_work_geometry(&parameters("3 cm")).unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(failed.axis(FeatureId(1)).is_ok());
    assert!(matches!(
        failed.health[2].status,
        oblikovati_model::feature::FeatureEvaluationStatus::Failed(_)
    ));
    assert_eq!(frame(&program, "7 cm").normal(), [0.0, 0.0, -1.0]);
}
#[test]
fn wrong_driver_kinds_and_suppression_do_not_provide_last_good_frames() {
    let mut program = program();
    let wrong = source(
        WorkAxisReference::Feature(FeatureId(2)),
        WorkPointReference::Origin,
    );
    program.replace(FeatureId(3), wrong, wrong.reads()).unwrap();
    let evaluation = program.evaluate_work_geometry(&parameters("5 cm")).unwrap();
    assert!(evaluation.plane(FeatureId(3)).is_err());
    assert!(format!("{:?}", evaluation.health[2]).contains("axis"));
    let mut program = program_with_suppressed_axis();
    program.set_suppressed(FeatureId(1), false).unwrap();
    assert_eq!(frame(&program, "5 cm").normal(), [0.0, 0.0, 1.0]);
}
fn program_with_suppressed_axis() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = program();
    program.set_suppressed(FeatureId(1), true).unwrap();
    assert!(
        program
            .evaluate_work_geometry(&parameters("5 cm"))
            .unwrap()
            .plane(FeatureId(3))
            .is_err()
    );
    program
}
#[test]
fn permanent_driver_reads_and_source_order_survive_versioned_round_trips() {
    let program = program();
    let recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 12);
    assert_eq!(recipe.features[2].definition.version, 7);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(program.same_work_geometry_recipe(&restored));
    let actual = recipe.features[2].definition.definition;
    let reads = program.get(FeatureId(3)).unwrap().definition().reads();
    let expected = [1, 2].map(|id| FeatureReadKey::Feature(FeatureId(id)));
    assert_eq!(reads.keys().iter().copied().collect::<Vec<_>>(), expected);
    let mut legacy = recipe.clone();
    legacy.version = 11;
    assert!(FeatureSequence::from_work_geometry_recipe(&legacy).is_err());
    assert!(matches!(
        actual,
        oblikovati_model::feature::work_plane::WorkPlaneDefinitionRecipe::LineThroughPoint(_)
    ));
}
#[test]
fn invalid_dependencies_cannot_replace_the_authored_definition() {
    let mut program = program();
    let before = program.clone();
    for id in [3, 99] {
        let invalid = source(
            WorkAxisReference::Feature(FeatureId(id)),
            WorkPointReference::Origin,
        );
        assert!(
            program
                .replace(FeatureId(3), invalid, invalid.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&before));
    }
}
#[test]
fn unknown_source_fields_and_wrong_source_versions_are_refused() {
    let definition = source(WorkAxisReference::OriginX, WorkPointReference::Origin);
    let mut value = serde_json::to_value(definition.recipe()).unwrap();
    value["definition"]["LineThroughPoint"]["unknown"] = true.into();
    assert!(
        serde_json::from_value::<oblikovati_model::feature::work_plane::WorkPlaneRecipe>(value)
            .is_err()
    );
    let mut recipe = definition.recipe();
    recipe.version = 6;
    assert!(WorkGeometryDefinition::from_recipe(recipe).is_err());
    assert!(!definition.same_recipe(source(
        WorkAxisReference::OriginY,
        WorkPointReference::Origin
    )));
}
#[test]
fn checked_axis_components_remain_exact_and_large_origins_do_not_erase_direction() {
    let direction = [1.0 / 3.0_f64.sqrt(); 3];
    let frame = SketchPlane::through_line_and_point(
        [0.0; 3],
        direction,
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    assert_eq!(
        frame.recipe().x_axis.map(f64::to_bits),
        direction.map(f64::to_bits)
    );
}
#[test]
fn large_line_origins_do_not_erase_checked_direction() {
    let origin = [f64::MAX, 0.0, 0.0];
    let frame = SketchPlane::through_line_and_point(
        origin,
        [1.0, 0.0, 0.0],
        [f64::MAX, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    assert_eq!(frame.recipe().origin, origin);
    assert_eq!(frame.normal(), [0.0, 0.0, 1.0]);
}
#[test]
fn generated_extreme_and_near_collinear_scales_keep_defined_frames() {
    for height in [f64::from_bits(1), 1e-150, 1.0, 1e150, f64::MAX] {
        let frame = SketchPlane::through_line_and_point(
            [0.0; 3],
            [1.0, 0.0, 0.0],
            [0.0, height, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_eq!(frame.normal(), [0.0, 0.0, 1.0]);
        let near = SketchPlane::through_line_and_point(
            [0.0; 3],
            [1.0, 0.0, 0.0],
            [1.0, height, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_eq!(near.normal(), [0.0, 0.0, 1.0]);
    }
}
#[test]
fn line_frame_refuses_nonfinite_origins_points_nonunit_axes_and_collinearity() {
    for (origin, direction, point) in [
        ([f64::NAN, 0.0, 0.0], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        ([0.0; 3], [0.0; 3], [0.0, 1.0, 0.0]),
        ([0.0; 3], [1.0, 0.0, 0.0], [f64::INFINITY, 0.0, 0.0]),
        ([0.0; 3], [1.0, 0.0, 0.0], [2.0, 0.0, 0.0]),
    ] {
        assert!(
            SketchPlane::through_line_and_point(origin, direction, point, Default::default())
                .is_err()
        );
    }
}
