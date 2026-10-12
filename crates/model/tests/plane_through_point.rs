// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureEvaluationStatus, FeatureId, FeatureReadKey, FeatureSequence,
        work_geometry::{
            DatumLength, WorkGeometryDefinition, WorkPlaneThroughPoint, WorkPointReference,
        },
        work_plane::{WorkPlaneOrigin, WorkPlaneReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

fn source(base: WorkPlaneReference, point: WorkPointReference) -> WorkGeometryDefinition {
    WorkGeometryDefinition::ParallelThroughPoint(WorkPlaneThroughPoint { base, point })
}
fn append(
    sequence: &mut FeatureSequence<WorkGeometryDefinition>,
    id: u64,
    definition: WorkGeometryDefinition,
) {
    sequence
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    append(
        &mut program,
        1,
        WorkGeometryDefinition::Fixed(SketchPlane::xz()),
    );
    let point = [
        DatumLength::Constant(5.0),
        DatumLength::Constant(6.0),
        DatumLength::Parameter(1),
    ];
    append(&mut program, 2, WorkGeometryDefinition::Point(point));
    let definition = source(
        WorkPlaneReference::Feature(FeatureId(1)),
        WorkPointReference::Feature(FeatureId(2)),
    );
    append(&mut program, 3, definition);
    program
}
fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "height", USER_PARAM, "7 cm")
        .unwrap();
    parameters
}
fn frame(
    program: &FeatureSequence<WorkGeometryDefinition>,
    parameters: &ParameterCollection<u64>,
) -> SketchPlane {
    program
        .evaluate_work_geometry(parameters)
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
#[test]
fn all_origin_orientations_place_the_plane_at_the_current_point() {
    let mut program = program();
    for origin in [
        WorkPlaneOrigin::XY,
        WorkPlaneOrigin::XZ,
        WorkPlaneOrigin::YZ,
    ] {
        let definition = source(
            WorkPlaneReference::Origin(origin),
            WorkPointReference::Feature(FeatureId(2)),
        );
        program
            .replace(FeatureId(3), definition.clone(), definition.reads())
            .unwrap();
        let recipe = frame(&program, &parameters()).recipe();
        assert_eq!(recipe.origin, [5.0, 6.0, 7.0]);
        assert_eq!(recipe.x_axis, origin.plane().recipe().x_axis);
        assert_eq!(recipe.y_axis, origin.plane().recipe().y_axis);
    }
}
#[test]
fn parameter_edits_and_base_orientation_remain_live() {
    let mut program = program();
    let mut parameters = parameters();
    parameters.set_expression(1, "11 cm").unwrap();
    assert_eq!(
        frame(&program, &parameters).recipe().origin,
        [5.0, 6.0, 11.0]
    );
    program.flip_plane_normal(FeatureId(1)).unwrap();
    let expected = SketchPlane::xz().flipped_normal().unwrap();
    assert_eq!(
        frame(&program, &parameters).recipe().y_axis,
        expected.recipe().y_axis
    );
    program.flip_plane_normal(FeatureId(3)).unwrap();
    assert_eq!(
        frame(&program, &parameters).recipe().y_axis,
        SketchPlane::xz().recipe().y_axis
    );
}
#[test]
fn reads_name_both_stable_drivers() {
    let program = program();
    let reads = program.get(FeatureId(3)).unwrap().reads().keys();
    let expected = [
        FeatureReadKey::Feature(FeatureId(1)),
        FeatureReadKey::Feature(FeatureId(2)),
    ];
    assert_eq!(reads.iter().copied().collect::<Vec<_>>(), expected);
}
#[test]
fn origin_drivers_have_no_feature_reads_and_resolve_the_document_origin() {
    let program = program();
    let definition = source(
        WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        WorkPointReference::Origin,
    );
    assert!(definition.reads().keys().is_empty());
    let current = program.evaluate_work_geometry(&parameters()).unwrap();
    assert!(
        definition
            .evaluate(&parameters(), &current)
            .unwrap()
            .same_recipe(SketchPlane::xy())
    );
}
#[test]
fn failed_point_retains_source_has_no_stale_frame_and_recovers() {
    let program = program();
    let mut parameters = parameters();
    parameters.set_expression(1, "missing").unwrap();
    let failed = program.evaluate_work_geometry(&parameters).unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(failed.plane(FeatureId(1)).is_ok());
    parameters.set_expression(1, "9 cm").unwrap();
    assert_eq!(
        frame(&program, &parameters).recipe().origin,
        [5.0, 6.0, 9.0]
    );
}
#[test]
fn new_source_requires_definition_five_and_program_ten() {
    let program = program();
    let mut recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 10);
    assert_eq!(recipe.features[2].definition.version, 5);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(program.same_work_geometry_recipe(&restored));
    recipe.version = 9;
    assert!(FeatureSequence::from_work_geometry_recipe(&recipe).is_err());
    recipe.version = 10;
    recipe.features[2].definition.version = 1;
    assert!(FeatureSequence::from_work_geometry_recipe(&recipe).is_err());
}
#[test]
fn changing_either_reference_changes_the_authored_recipe() {
    let definition = program().get(FeatureId(3)).unwrap().definition().clone();
    assert!(definition.same_recipe(&definition));
    assert!(!definition.same_recipe(&source(
        WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        WorkPointReference::Feature(FeatureId(2))
    )));
    assert!(!definition.same_recipe(&source(
        WorkPlaneReference::Feature(FeatureId(1)),
        WorkPointReference::Origin
    )));
}
#[test]
fn wrong_driver_kinds_are_named_declines() {
    let mut program = program();
    for (invalid, expected) in wrong_driver_sources() {
        program
            .replace(FeatureId(3), invalid.clone(), invalid.reads())
            .unwrap();
        assert!(decline_reason(&program).contains(expected));
    }
}
fn wrong_driver_sources() -> [(WorkGeometryDefinition, &'static str); 2] {
    let invalid_base = source(
        WorkPlaneReference::Feature(FeatureId(2)),
        WorkPointReference::Origin,
    );
    let invalid_point = source(
        WorkPlaneReference::Feature(FeatureId(1)),
        WorkPointReference::Feature(FeatureId(1)),
    );
    [
        (invalid_base, "expected an available current plane"),
        (invalid_point, "expected an available current point"),
    ]
}
#[test]
fn self_and_forward_dependencies_cannot_replace_the_authored_source() {
    let mut program = program();
    let before = program.clone();
    for id in [3, 99] {
        let invalid = source(
            WorkPlaneReference::Feature(FeatureId(id)),
            WorkPointReference::Origin,
        );
        assert!(
            program
                .replace(FeatureId(3), invalid.clone(), invalid.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&before));
    }
}
fn decline_reason(program: &FeatureSequence<WorkGeometryDefinition>) -> String {
    let evaluation = program.evaluate_work_geometry(&parameters()).unwrap();
    assert!(evaluation.plane(FeatureId(3)).is_err());
    let FeatureEvaluationStatus::Failed(cause) = &evaluation.health[2].status else {
        panic!("expected a failed plane with a typed cause");
    };
    cause.to_string()
}
#[test]
fn unknown_stored_source_fields_are_refused() {
    let definition = program().get(FeatureId(3)).unwrap().definition().clone();
    let mut value = serde_json::to_value(definition.recipe()).unwrap();
    value["definition"]["ParallelThroughPoint"]["unknown"] = true.into();
    assert!(
        serde_json::from_value::<oblikovati_model::feature::work_plane::WorkPlaneRecipe>(value)
            .is_err()
    );
}
