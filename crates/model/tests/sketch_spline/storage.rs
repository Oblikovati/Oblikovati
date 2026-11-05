// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn schema_ten_preserves_parent_intent_signed_zero_and_shared_identity() {
    for kind in spline_kinds() {
        assert_saved_parent(kind);
    }
}
fn assert_saved_parent(kind: SketchSplineKind) {
    let sketch = owned(kind, -0.0);
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 10);
    let serialized = serde_json::to_string(&recipe).unwrap();
    let decoded: SketchRecipe = serde_json::from_str(&serialized).unwrap();
    let restored = Sketch2D::from_recipe(&decoded).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_saved_spline_identity(&restored, &sketch, kind);
}
fn assert_saved_spline_identity(restored: &Sketch2D, sketch: &Sketch2D, kind: SketchSplineKind) {
    assert_eq!(
        restored.spline(id(4)).unwrap().points(),
        &[id(1), id(2), id(3)]
    );
    assert_eq!(
        restored.offset_spline(id(5)).unwrap().distance().to_bits(),
        (-0.0f64).to_bits()
    );
    assert!(!sketch.same_recipe(&owned(kind, 0.0)));
    assert_eq!(restored.splines().len(), 1);
    assert_eq!(restored.offset_splines().len(), 1);
}

#[test]
fn older_schemas_refuse_spline_fields_and_schema_ten_requires_explicit_lists() {
    let recipe = owned(SketchSplineKind::Control, 1.0).recipe();
    for version in 1..=9 {
        let mut legacy = recipe.clone();
        legacy.version = version;
        assert!(
            Sketch2D::from_recipe(&legacy)
                .unwrap_err()
                .to_string()
                .contains("versions 10-12")
        );
    }
    assert_missing_spline_lists(recipe);
}
fn assert_missing_spline_lists(recipe: SketchRecipe) {
    let mut incomplete = recipe.clone();
    incomplete.splines = None;
    assert!(Sketch2D::from_recipe(&incomplete).is_err());
    incomplete = recipe;
    incomplete.offset_splines = None;
    assert!(Sketch2D::from_recipe(&incomplete).is_err());
    assert_eq!(
        Sketch2D::from_recipe(&Sketch2D::default().recipe()).unwrap(),
        Sketch2D::default()
    );
}
#[test]
fn saved_wrong_parent_unknown_fit_method_and_nonfinite_distance_are_refused() {
    for field in ["parent", "distance", "id"] {
        assert_invalid_saved_offset(field);
    }
    let mut recipe = owned(SketchSplineKind::Control, 1.0).recipe();
    let serialized = serde_json::to_string(&recipe.splines.as_ref().unwrap()[0]).unwrap();
    let invalid = serialized.replace("\"Control\"", "{\"Fit\":{\"method\":-1}}");
    let parent: SketchSpline2D = serde_json::from_str(&invalid).unwrap();
    recipe.splines = Some(vec![parent]);
    assert!(
        Sketch2D::from_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("fit method")
    );
}
fn assert_invalid_saved_offset(field: &str) {
    let mut recipe = owned(SketchSplineKind::Control, 1.0).recipe();
    let mut value = serde_json::to_value(recipe.offset_splines.as_ref().unwrap()[0]).unwrap();
    value[field] = serde_json::json!(0);
    if field == "distance" {
        value[field] = serde_json::json!("invalid");
    }
    let result = serde_json::from_value::<SketchOffsetSpline2D>(value);
    if field == "distance" {
        assert!(result.is_err());
        return;
    }
    recipe.offset_splines = Some(vec![result.unwrap()]);
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}
#[test]
fn prepared_curve_definitions_refuse_bad_identity_shape_and_method() {
    for (curve, points) in [
        (0, vec![id(1), id(2)]),
        (4, vec![]),
        (4, vec![id(1)]),
        (4, vec![id(0), id(2)]),
        (4, vec![id(1), id(4)]),
    ] {
        let error = SketchSpline2D::new(id(curve), points, false, SketchSplineKind::Control)
            .unwrap_err()
            .to_string();
        assert!(error.contains("expected"));
    }
    assert_bad_fit_and_offset_definitions();
}
fn assert_bad_fit_and_offset_definitions() {
    assert!(
        SketchSpline2D::new(
            id(4),
            vec![id(1), id(2)],
            false,
            SketchSplineKind::Fit {
                method: oblikovati_api::types::SplineFitMethod(-1)
            }
        )
        .is_err()
    );
    for (curve, parent, distance) in [(0, 4, 1.0), (5, 0, 1.0), (4, 4, 1.0), (5, 4, f64::INFINITY)]
    {
        assert!(SketchOffsetSpline2D::new(id(curve), id(parent), distance).is_err());
    }
}
