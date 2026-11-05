// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    feature::work_axis::{WorkAxisError, WorkAxisLine, WorkAxisLineRecipe},
    sketch::{SketchFrameAxis, SketchPlane, SketchPlaneOptions},
};
use std::error::Error;

fn assert_direction(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() <= 8.0 * f64::EPSILON,
            "direction {actual}: expected {expected}"
        );
    }
}
#[test]
fn line_axis_matches_go_origin_and_direction_and_preserves_the_source_value() {
    let origin = [1.0, 2.0, 3.0];
    let axis = WorkAxisLine::new(origin, [0.0, 0.0, 7.0]).unwrap();
    assert_eq!(axis.origin(), origin);
    assert_eq!(axis.direction(), [0.0, 0.0, 1.0]);
    assert_eq!(axis.recipe().version, 1);
    assert!(axis.same_recipe(WorkAxisLine::from_recipe(axis.recipe()).unwrap()));
}
#[test]
fn dimensionless_direction_scales_normalize_without_overflow_or_subnormal_loss() {
    let expected = [
        std::f64::consts::FRAC_1_SQRT_2,
        std::f64::consts::FRAC_1_SQRT_2,
        0.0,
    ];
    for scale in [f64::from_bits(1), f64::MIN_POSITIVE, 1.0, f64::MAX] {
        let axis = WorkAxisLine::new([0.0; 3], [scale, scale, 0.0]).unwrap();
        assert_direction(axis.direction(), expected);
        let opposite = WorkAxisLine::new([0.0; 3], [-scale, -scale, 0.0]).unwrap();
        assert_direction(opposite.direction(), expected.map(|value| -value));
    }
}
#[test]
fn invalid_origins_and_directions_return_their_named_causes() {
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let error = WorkAxisLine::new([value, 0.0, 0.0], [1.0, 0.0, 0.0]).unwrap_err();
        assert!(matches!(error, WorkAxisError::Origin(_)));
        assert!(error.to_string().contains("origin"));
        assert!(error.source().is_none());
        let error = WorkAxisLine::new([0.0; 3], [value, 1.0, 0.0]).unwrap_err();
        assert!(matches!(error, WorkAxisError::Direction { .. }));
        assert!(error.source().is_some());
        assert!(error.to_string().contains("expected"));
    }
    let error = WorkAxisLine::new([0.0; 3], [-0.0, 0.0, 0.0]).unwrap_err();
    assert!(error.to_string().contains("nonzero"));
}
#[test]
fn repeated_serialized_restore_preserves_direction_bits_and_signed_origins() {
    let original = WorkAxisLine::new([-0.0, 7.0, -11.0], [3.0, 5.0, 7.0]).unwrap();
    let mut current = original;
    for _ in 0..32 {
        let encoded = serde_json::to_vec(&current.recipe()).unwrap();
        let recipe: WorkAxisLineRecipe = serde_json::from_slice(&encoded).unwrap();
        current = WorkAxisLine::from_recipe(recipe).unwrap();
        assert!(original.same_recipe(current));
    }
    let positive = WorkAxisLine::new([0.0, 7.0, -11.0], [3.0, 5.0, 7.0]).unwrap();
    assert!(!positive.same_recipe(original));
    let reversed = WorkAxisLine::new(original.origin(), [-3.0, -5.0, -7.0]).unwrap();
    assert!(!original.same_recipe(reversed));
}
#[test]
fn checked_axis_direction_is_consumed_by_the_same_sketch_orientation_operation() {
    let axis = WorkAxisLine::new([1.0, 2.0, 3.0], [3.0, 4.0, 5.0]).unwrap();
    let oriented = SketchPlane::xy()
        .with_orientation(axis.direction(), SketchFrameAxis::Y, false, None)
        .unwrap();
    assert_direction(oriented.recipe().y_axis, [0.6, 0.8, 0.0]);
    assert_direction(oriented.normal(), [0.0, 0.0, 1.0]);
}
#[test]
fn malformed_versions_and_unchecked_saved_unit_vectors_are_refused() {
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    let mut recipe = axis.recipe();
    recipe.version = 2;
    let error = WorkAxisLine::from_recipe(recipe).unwrap_err();
    assert_eq!(error, WorkAxisError::RecipeVersion(2));
    assert!(error.to_string().contains("expected version 1"));
    assert!(error.source().is_none());
}
#[test]
fn invalid_saved_directions_and_origins_are_refused_separately() {
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    let mut recipe = axis.recipe();
    recipe.version = 1;
    for direction in [
        [0.0; 3],
        [2.0, 0.0, 0.0],
        [f64::MAX; 3],
        [f64::NAN, 0.0, 0.0],
    ] {
        recipe.direction = direction;
        assert!(WorkAxisLine::from_recipe(recipe).is_err());
    }
    recipe.direction = [1.0, 0.0, 0.0];
    recipe.origin = [f64::INFINITY; 3];
    assert!(matches!(
        WorkAxisLine::from_recipe(recipe),
        Err(WorkAxisError::Origin(_))
    ));
}
#[test]
fn restore_checks_the_existing_unit_metric_policy_without_rewriting_components() {
    let deviation = SketchPlaneOptions::default().unit_deviation;
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    let mut recipe = axis.recipe();
    recipe.direction[0] += deviation / 2.0;
    assert_eq!(WorkAxisLine::from_recipe(recipe).unwrap().recipe(), recipe);
    recipe.direction[0] = 1.0 + deviation * 2.0;
    assert!(WorkAxisLine::from_recipe(recipe).is_err());
    assert!(
        serde_json::from_str::<WorkAxisLineRecipe>(
            r#"{"version":1,"origin":[0,0,0],"direction":[1,0,0],"cache":true}"#
        )
        .is_err()
    );
}
