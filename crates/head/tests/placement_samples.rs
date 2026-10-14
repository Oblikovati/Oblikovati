// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    PlacementPreviewCurve, SketchDisplayEntity, SketchDisplayPrimitive, SketchSamplingOptionsV1,
    sample_placement_preview, sample_sketch_display,
};
use oblikovati_model::sketch::{Sketch2D, SketchPlacementRecipe};
pub(crate) fn assert_recipe_tail(geometry: &Sketch2D, recipe: &SketchPlacementRecipe) {
    let options = SketchSamplingOptionsV1 {
        deflection: 0.01,
        maximum_curve_points: 4096,
    };
    let preview = sample_placement_preview(recipe, options).unwrap();
    let display = sample_sketch_display(geometry, options).unwrap();
    assert!(display.declines.is_empty(), "{:?}", display.declines);
    assert!(display.entities.len() >= preview.len());
    let tail = &display.entities[display.entities.len() - preview.len()..];
    for (actual, expected) in tail.iter().zip(&preview) {
        assert_curve(actual, expected);
    }
}
fn assert_curve(actual: &SketchDisplayEntity, expected: &PlacementPreviewCurve) {
    let SketchDisplayPrimitive::Curve(points) = &actual.primitive else {
        panic!("expected curve samples")
    };
    assert_eq!(actual.construction, expected.construction);
    assert_eq!(points.len(), expected.points.len());
    for (a, b) in points
        .iter()
        .flatten()
        .zip(expected.points.iter().flatten())
    {
        assert!((a - b).abs() < 1e-8, "committed {a} != preview {b}");
    }
}
