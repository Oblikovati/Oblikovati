// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    PlacementPreviewMember, SketchDisplayEntity, SketchDisplayPrimitive, SketchSamplingOptionsV1,
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
        assert_member(actual, expected);
    }
}
fn assert_member(actual: &SketchDisplayEntity, expected: &PlacementPreviewMember) {
    assert_eq!(actual.construction, expected.construction);
    match (&actual.primitive, &expected.primitive) {
        (SketchDisplayPrimitive::Point(a), SketchDisplayPrimitive::Point(b)) => assert_eq!(a, b),
        (SketchDisplayPrimitive::Curve(a), SketchDisplayPrimitive::Curve(b)) => {
            assert_curve_samples(a, b)
        }
        pair => panic!("expected matching display primitives, got {pair:?}"),
    }
}
fn assert_curve_samples(actual: &[[f64; 2]], expected: &[[f64; 2]]) {
    assert_eq!(actual.len(), expected.len());
    for (a, b) in actual.iter().flatten().zip(expected.iter().flatten()) {
        assert!((a - b).abs() < 1e-8, "committed {a} != preview {b}");
    }
}
