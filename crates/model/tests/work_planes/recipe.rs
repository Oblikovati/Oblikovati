// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn fixed_recipe_round_trip_preserves_authored_signed_zero_and_axes() {
    let frame = SketchPlane::new(
        [-0.0, 2.0, 3.0],
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    let definition = WorkPlaneDefinition::Fixed(frame);
    let json = serde_json::to_string(&definition.recipe()).unwrap();
    let recipe = serde_json::from_str(&json).unwrap();
    let restored = WorkPlaneDefinition::from_recipe(recipe).unwrap();
    let result = restored
        .evaluate(&parameters(0.0, Dimension::Length), &UnavailablePlanes)
        .unwrap();
    assert!(frame.same_recipe(result));
    assert!(definition.reads().keys().is_empty());
}

#[test]
fn offset_recipe_retains_parameter_and_stable_feature_reference() {
    let definition = offset(
        WorkPlaneReference::Feature(FeatureId(7)),
        WorkPlaneDistance::Parameter(9),
    );
    let json = serde_json::to_string(&definition.recipe()).unwrap();
    let restored = WorkPlaneDefinition::from_recipe(serde_json::from_str(&json).unwrap()).unwrap();
    assert_eq!(restored, definition);
    assert_eq!(restored.reads().keys().len(), 2);
}

#[test]
fn offset_recipe_preserves_signed_zero_distance() {
    let constant = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Constant(-0.0));
    let json = serde_json::to_string(&constant.recipe()).unwrap();
    let restored = WorkPlaneDefinition::from_recipe(serde_json::from_str(&json).unwrap()).unwrap();
    let WorkPlaneDefinition::Offset {
        distance: WorkPlaneDistance::Constant(value),
        ..
    } = restored
    else {
        panic!("expected constant offset");
    };
    assert_eq!(value.to_bits(), (-0.0_f64).to_bits());
}

#[test]
fn recipes_reject_versions_nonfinite_offsets_and_invalid_frames() {
    let mut recipe = WorkPlaneDefinition::Fixed(SketchPlane::xy()).recipe();
    recipe.version = 2;
    assert_eq!(
        WorkPlaneDefinition::from_recipe(recipe),
        Err(WorkPlaneError::RecipeVersion(2))
    );
    assert!(
        WorkPlaneError::RecipeVersion(2)
            .to_string()
            .contains("version 2: expected 1")
    );
}

#[test]
fn restored_constant_offsets_must_be_finite() {
    let bad = origin_offset(
        WorkPlaneOrigin::XY,
        WorkPlaneDistance::Constant(f64::INFINITY),
    )
    .recipe();
    assert!(matches!(
        WorkPlaneDefinition::from_recipe(bad),
        Err(WorkPlaneError::Offset { .. })
    ));
}

#[test]
fn fixed_recipe_refuses_invalid_authored_axes() {
    let mut bad = WorkPlaneDefinition::Fixed(SketchPlane::xy()).recipe();
    let WorkPlaneDefinitionRecipe::Fixed { frame, .. } = &mut bad.definition else {
        unreachable!();
    };
    frame.x_axis = [0.0; 3];
    assert!(matches!(
        WorkPlaneDefinition::from_recipe(bad),
        Err(WorkPlaneError::Frame(_))
    ));
}

#[test]
fn serialized_recipes_refuse_unknown_fields_and_definition_kinds() {
    let recipe = WorkPlaneDefinition::Fixed(SketchPlane::xy()).recipe();
    let mut json = serde_json::to_value(&recipe).unwrap();
    json["extra"] = true.into();
    assert!(serde_json::from_value::<WorkPlaneRecipe>(json).is_err());
    let mut json = serde_json::to_value(&recipe).unwrap();
    json["definition"]["Fixed"]["extra"] = true.into();
    assert!(serde_json::from_value::<WorkPlaneRecipe>(json).is_err());
    let mut json = serde_json::to_value(&recipe).unwrap();
    json["definition"] = serde_json::json!({"Unsupported": {}});
    assert!(serde_json::from_value::<WorkPlaneRecipe>(json).is_err());
}
