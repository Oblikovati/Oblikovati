// SPDX-License-Identifier: GPL-2.0-only

use super::*;
mod comparison;
mod construction;

fn authored_program() -> FeatureSequence<WorkPlaneDefinition> {
    let mut sequence = chain();
    let keys = [
        FeatureReadKey::Feature(FeatureId(1)),
        FeatureReadKey::Sketch(9),
    ];
    let reads = FeatureReadFootprint::new(keys, true);
    sequence
        .replace(FeatureId(2), referenced_offset(1, -2.0), reads)
        .unwrap();
    sequence.rename(FeatureId(2), "Relative").unwrap();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    let condition = length_condition(1, FeatureSuppressionComparison::LessThan, 20.0);
    sequence
        .set_suppression_condition(FeatureId(3), Some(condition))
        .unwrap();
    sequence.set_end_of_part(Some(3)).unwrap();
    sequence
}

fn length_condition(
    parameter: u64,
    comparison: FeatureSuppressionComparison,
    value: f64,
) -> FeatureSuppressionCondition {
    let threshold = Quantity::new(value, Dimension::Length);
    FeatureSuppressionCondition::new(parameter, comparison, threshold).unwrap()
}

fn restored(
    sequence: &FeatureSequence<WorkPlaneDefinition>,
) -> FeatureSequence<WorkPlaneDefinition> {
    let json = serde_json::to_string(&sequence.work_plane_recipe().unwrap()).unwrap();
    let recipe = serde_json::from_str(&json).unwrap();
    FeatureSequence::from_work_plane_recipe(&recipe).unwrap()
}

#[test]
fn full_program_round_trip_preserves_authored_state_reads_and_real_geometry() {
    let sequence = authored_program();
    let restored = restored(&sequence);
    assert!(sequence.same_work_plane_recipe(&restored));
    let parameters = parameters(5.0, Dimension::Length);
    let first = sequence.evaluate_work_planes(&parameters).unwrap();
    let second = restored.evaluate_work_planes(&parameters).unwrap();
    assert_eq!(first.health, second.health);
    for id in 1..=3 {
        assert_eq!(first.plane(FeatureId(id)), second.plane(FeatureId(id)));
        assert_eq!(
            sequence.get(FeatureId(id)).unwrap().reads(),
            restored.get(FeatureId(id)).unwrap().reads()
        );
    }
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(99)]),
        restored.replay_for_reads([FeatureReadKey::Parameter(99)])
    );
}

#[test]
fn restored_condition_reads_can_be_cleared_without_changing_definition_reads() {
    let mut sequence = restored(&authored_program());
    let first = sequence.get(FeatureId(3)).unwrap();
    assert!(first.reads().keys().contains(&FeatureReadKey::Parameter(1)));
    sequence
        .set_suppression_condition(FeatureId(3), None)
        .unwrap();
    let cleared = sequence.get(FeatureId(3)).unwrap();
    assert!(cleared.reads().keys().is_empty());
    let unattributed = sequence.get(FeatureId(2)).unwrap();
    assert!(unattributed.reads().has_unattributed_reads());
}

#[test]
fn clamped_authored_cutoff_survives_round_trip_and_a_new_tail_entry() {
    let mut sequence = authored_program();
    sequence.remove(FeatureId(3)).unwrap();
    let mut restored = restored(&sequence);
    assert_eq!(restored.end_of_part(), Some(3));
    assert_eq!(restored.effective_end(), 2);
    assert!(sequence.same_work_plane_recipe(&restored));
    add(
        &mut restored,
        9,
        WorkPlaneDefinition::Fixed(SketchPlane::xz()),
    );
    assert_eq!(restored.effective_end(), 3);
    let report = restored
        .evaluate_work_planes(&parameters(5.0, Dimension::Length))
        .unwrap();
    assert!(report.plane(FeatureId(9)).is_ok());
}

#[test]
fn empty_program_round_trip_is_independent_and_has_no_derived_frames() {
    let sequence = FeatureSequence::<WorkPlaneDefinition>::default();
    let mut restored = restored(&sequence);
    assert!(sequence.same_work_plane_recipe(&restored));
    add(
        &mut restored,
        1,
        WorkPlaneDefinition::Fixed(SketchPlane::xy()),
    );
    assert!(sequence.features().is_empty());
    assert!(!sequence.same_work_plane_recipe(&restored));
}

#[test]
fn duplicate_and_forward_ids_cannot_restore_over_a_valid_recipe() {
    let sequence = authored_program();
    let mut duplicate = sequence.work_plane_recipe().unwrap();
    duplicate.features[1].id = FeatureId(1);
    let error = FeatureSequence::from_work_plane_recipe(&duplicate).unwrap_err();
    assert!(matches!(error, WorkPlaneProgramRecipeError::Sequence(_)));
    assert!(error.source().is_some());
    assert!(error.to_string().contains("expected an unused ID"));
    let mut forward = sequence.work_plane_recipe().unwrap();
    forward.features.swap(0, 1);
    assert!(matches!(
        FeatureSequence::from_work_plane_recipe(&forward),
        Err(WorkPlaneProgramRecipeError::Sequence(_))
    ));
    assert!(sequence.same_work_plane_recipe(&restored(&sequence)));
}

#[test]
fn versions_and_invalid_definitions_are_named_restore_failures() {
    let mut recipe = authored_program().work_plane_recipe().unwrap();
    recipe.version = 15;
    let error = FeatureSequence::from_work_plane_recipe(&recipe).unwrap_err();
    assert_eq!(error, WorkPlaneProgramRecipeError::Version(15));
    assert_eq!(
        error.to_string(),
        "work plane program version 15: expected 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, or 14"
    );
    assert!(error.source().is_none());
    recipe.version = 1;
    recipe.features[0].definition.version = 2;
    let error = FeatureSequence::from_work_plane_recipe(&recipe).unwrap_err();
    assert!(matches!(error, WorkPlaneProgramRecipeError::Plane(_)));
    assert!(error.source().is_some());
    assert!(error.to_string().contains("recipe version 2: expected 1"));
}

#[test]
fn missing_declared_reads_are_refused_on_restore_and_export() {
    let mut recipe = authored_program().work_plane_recipe().unwrap();
    recipe.features[0].definition_reads = FeatureReadFootprint::default();
    let error = FeatureSequence::from_work_plane_recipe(&recipe).unwrap_err();
    assert!(matches!(error, WorkPlaneProgramRecipeError::Reads(_)));
    assert!(error.to_string().contains("missing reads [Parameter(1)]"));
    assert!(error.source().is_some());
    assert!(matches!(
        undeclared_program().work_plane_recipe(),
        Err(WorkPlaneProgramRecipeError::Reads(_))
    ));
}

#[test]
fn invalid_constant_definitions_cannot_be_exported() {
    let mut sequence = FeatureSequence::default();
    add(
        &mut sequence,
        1,
        origin_offset(
            WorkPlaneOrigin::XY,
            WorkPlaneDistance::Constant(f64::INFINITY),
        ),
    );
    assert!(matches!(
        sequence.work_plane_recipe(),
        Err(WorkPlaneProgramRecipeError::Plane(_))
    ));
}

fn invalid_category(category: &str) -> WorkPlaneProgramRecipeError {
    let mut recipe = authored_program().work_plane_recipe().unwrap();
    recipe.features[2].condition.as_mut().unwrap().category = category.into();
    FeatureSequence::from_work_plane_recipe(&recipe).unwrap_err()
}

#[test]
fn condition_categories_are_canonical_and_numeric() {
    let error = invalid_category("mm");
    assert!(matches!(
        error,
        WorkPlaneProgramRecipeError::Condition(FeatureSuppressionError::Category { .. })
    ));
    assert!(
        error
            .to_string()
            .contains("parameter 1 category \"mm\": expected a canonical quantity category")
    );
    assert!(error.source().is_some());
}

#[test]
fn known_nonnumeric_condition_categories_cannot_restore() {
    let error = invalid_category("boolean");
    assert!(matches!(
        error,
        WorkPlaneProgramRecipeError::Condition(FeatureSuppressionError::Threshold { .. })
    ));
    assert!(
        error
            .to_string()
            .contains("expected a finite numeric working-unit quantity")
    );
}

#[test]
fn malformed_numeric_conditions_are_refused() {
    let mut recipe = authored_program().work_plane_recipe().unwrap();
    recipe.features[2].condition.as_mut().unwrap().threshold = f64::NAN;
    assert!(matches!(
        FeatureSequence::from_work_plane_recipe(&recipe),
        Err(WorkPlaneProgramRecipeError::Condition(
            FeatureSuppressionError::Threshold { .. }
        ))
    ));
}

#[test]
fn all_recipe_levels_refuse_unknown_fields() {
    for pointer in [
        "",
        "/features/0",
        "/features/0/definition_reads",
        "/features/2/condition",
    ] {
        let mut json =
            serde_json::to_value(authored_program().work_plane_recipe().unwrap()).unwrap();
        json.pointer_mut(pointer)
            .unwrap()
            .as_object_mut()
            .unwrap()
            .insert("extra".into(), true.into());
        assert!(
            serde_json::from_value::<WorkPlaneProgramRecipe>(json).is_err(),
            "pointer={pointer}"
        );
    }
}

#[test]
fn unknown_comparisons_and_read_keys_are_refused() {
    let recipe = authored_program().work_plane_recipe().unwrap();
    let mut json = serde_json::to_value(&recipe).unwrap();
    json["features"][2]["condition"]["comparison"] = "Unsupported".into();
    assert!(serde_json::from_value::<WorkPlaneProgramRecipe>(json).is_err());
    let mut json = serde_json::to_value(&recipe).unwrap();
    json["features"][0]["definition_reads"]["keys"] = serde_json::json!([{"Unsupported": 1}]);
    assert!(serde_json::from_value::<WorkPlaneProgramRecipe>(json).is_err());
}

#[test]
fn reordered_readers_keep_the_same_removal_refusal_after_restore() {
    let mut sequence = FeatureSequence::default();
    add(
        &mut sequence,
        10,
        WorkPlaneDefinition::Fixed(SketchPlane::xy()),
    );
    add(&mut sequence, 7, referenced_offset(10, 1.0));
    add(&mut sequence, 2, referenced_offset(10, 2.0));
    sequence.reorder(FeatureId(2), 1).unwrap();
    let mut restored = restored(&sequence);
    assert_eq!(
        sequence.remove(FeatureId(10)),
        restored.remove(FeatureId(10))
    );
}
