// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn construction_changes_only_lifecycle_state_and_exact_recipe_comparison() {
    let original = chain();
    let mut changed = original.clone();
    assert!(!changed.get(FeatureId(1)).unwrap().is_construction_datum());
    changed
        .set_work_plane_construction(FeatureId(1), true)
        .unwrap();
    assert!(changed.get(FeatureId(1)).unwrap().is_construction_datum());
    assert!(!original.same_work_plane_recipe(&changed));
    assert_same_frames(&original, &changed);
    changed
        .set_work_plane_construction(FeatureId(1), false)
        .unwrap();
    assert!(original.same_work_plane_recipe(&changed));
}

#[test]
fn construction_recipes_use_version_two_and_round_trip_stable_identity() {
    let mut sequence = authored_program();
    sequence
        .set_work_plane_construction(FeatureId(2), true)
        .unwrap();
    let recipe = sequence.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 2);
    assert!(recipe.features[1].construction);
    let encoded = serde_json::to_string(&recipe).unwrap();
    let decoded: WorkPlaneProgramRecipe = serde_json::from_str(&encoded).unwrap();
    let restored = FeatureSequence::from_work_plane_recipe(&decoded).unwrap();
    assert!(sequence.same_work_plane_recipe(&restored));
    assert_eq!(restored.work_plane_recipe().unwrap(), recipe);
}

#[test]
fn ordinary_version_one_recipes_keep_their_bytes_and_omit_the_new_flag() {
    let mut sequence = authored_program();
    let original = serde_json::to_string(&sequence.work_plane_recipe().unwrap()).unwrap();
    assert!(!original.contains("\"construction\""));
    let decoded: WorkPlaneProgramRecipe = serde_json::from_str(&original).unwrap();
    assert_eq!(decoded.version, 1);
    assert!(decoded.features.iter().all(|feature| !feature.construction));
    sequence
        .set_work_plane_construction(FeatureId(2), true)
        .unwrap();
    sequence
        .set_work_plane_construction(FeatureId(2), false)
        .unwrap();
    assert_eq!(
        serde_json::to_string(&sequence.work_plane_recipe().unwrap()).unwrap(),
        original
    );
}

#[test]
fn construction_true_in_a_legacy_recipe_is_a_named_decline() {
    let mut recipe = chain().work_plane_recipe().unwrap();
    recipe.features[0].construction = true;
    let error = FeatureSequence::from_work_plane_recipe(&recipe).unwrap_err();
    assert_eq!(
        error,
        WorkPlaneProgramRecipeError::LegacyConstruction(FeatureId(1))
    );
    let message = error.to_string();
    assert!(message.contains("work plane 1"));
    assert!(message.contains("construction=true"));
    assert!(message.contains("expected version 2"));
    assert!(error.source().is_none());
}

#[test]
fn missing_feature_flags_are_rejected_without_changing_the_program() {
    let mut sequence = chain();
    let original = sequence.clone();
    for id in [0, 999] {
        let error = sequence
            .set_work_plane_construction(FeatureId(id), true)
            .unwrap_err();
        assert!(error.to_string().contains(&id.to_string()));
        assert!(sequence.same_work_plane_recipe(&original));
    }
}

#[test]
fn replacement_order_and_suppression_preserve_the_flag_on_its_identity() {
    let mut sequence = chain();
    let id = FeatureId(1);
    sequence.set_work_plane_construction(id, true).unwrap();
    let definition = WorkPlaneDefinition::Fixed(SketchPlane::xz());
    sequence
        .replace(id, definition, definition.reads())
        .unwrap();
    sequence.rename(id, "Construction base").unwrap();
    sequence.set_suppressed(id, true).unwrap();
    sequence.reorder(FeatureId(3), 0).unwrap();
    let restored = restored(&sequence);
    assert!(restored.get(id).unwrap().is_construction_datum());
    assert!(sequence.same_work_plane_recipe(&restored));
}

fn assert_same_frames(
    original: &FeatureSequence<WorkPlaneDefinition>,
    changed: &FeatureSequence<WorkPlaneDefinition>,
) {
    let parameters = parameters(5.0, Dimension::Length);
    let before = original.evaluate_work_planes(&parameters).unwrap();
    let after = changed.evaluate_work_planes(&parameters).unwrap();
    for id in [1, 2, 3] {
        assert!(
            before
                .plane(FeatureId(id))
                .unwrap()
                .same_recipe(after.plane(FeatureId(id)).unwrap())
        );
    }
}

#[test]
fn version_two_without_flags_preserves_ordinary_plane_state() {
    let mut recipe = chain().work_plane_recipe().unwrap();
    recipe.version = 2;
    let restored = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert!(
        restored
            .features()
            .iter()
            .all(|feature| !feature.is_construction_datum())
    );
    assert_eq!(restored.work_plane_recipe().unwrap().version, 1);
}
