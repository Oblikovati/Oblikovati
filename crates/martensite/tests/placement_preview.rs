// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    PlacementPreviewCurve, SketchSamplingOptionsV1, sample_placement_preview,
};
use oblikovati_model::sketch::{SketchPathMember, SketchPlacementRecipe};
fn options(deflection: f64) -> SketchSamplingOptionsV1 {
    SketchSamplingOptionsV1 {
        deflection,
        maximum_curve_points: 4096,
    }
}
#[test]
fn preview_preserves_order_roles_endpoints_and_curve_error() {
    let straight = SketchPlacementRecipe::straight_slot([[3., 4.], [-7., 8.]], 2.).unwrap();
    for recipe in [straight, arc(true), arc(false)] {
        verify_recipe(&recipe);
    }
}
fn verify_recipe(recipe: &SketchPlacementRecipe) {
    let original = recipe.clone();
    let curves = sample_placement_preview(recipe, options(0.01)).unwrap();
    assert_eq!(
        curves,
        sample_placement_preview(recipe, options(0.01)).unwrap()
    );
    assert_eq!(*recipe, original);
    assert_eq!(curves.len(), recipe.members.len());
    for (index, curve) in curves.iter().enumerate() {
        verify_member(recipe, index, curve);
    }
    let fine = sample_placement_preview(recipe, options(0.001)).unwrap();
    assert!(
        fine.iter().map(|c| c.points.len()).sum::<usize>()
            > curves.iter().map(|c| c.points.len()).sum()
    );
}
fn verify_member(recipe: &SketchPlacementRecipe, index: usize, curve: &PlacementPreviewCurve) {
    let member = recipe.members[index];
    assert_eq!(curve.member_index, index);
    assert_eq!(curve.construction, member.construction);
    let endpoints = match member.path {
        SketchPathMember::Line(ends) => ends,
        SketchPathMember::Arc { endpoints, .. } => endpoints,
    };
    for (sample, id) in [curve.points.first().unwrap(), curve.points.last().unwrap()]
        .into_iter()
        .zip(endpoints)
    {
        assert!(distance(*sample, recipe.points[id]) < 1e-12);
    }
    if let SketchPathMember::Arc {
        center,
        counter_clockwise,
        ..
    } = member.path
    {
        let center = recipe.points[center];
        verify_arc(
            curve,
            center,
            distance(center, recipe.points[endpoints[0]]),
            counter_clockwise,
        );
    } else {
        assert_eq!(curve.points.len(), 2);
    }
}
fn verify_arc(curve: &PlacementPreviewCurve, center: [f64; 2], radius: f64, ccw: bool) {
    for pair in curve.points.windows(2) {
        let midpoint = [
            (pair[0][0] + pair[1][0]) * 0.5,
            (pair[0][1] + pair[1][1]) * 0.5,
        ];
        assert!((radius - distance(center, midpoint)).abs() <= 0.01 + 1e-12);
        let a = [pair[0][0] - center[0], pair[0][1] - center[1]];
        let b = [pair[1][0] - center[0], pair[1][1] - center[1]];
        assert_eq!(a[0] * b[1] - a[1] * b[0] > 0., ccw);
    }
}
fn arc(ccw: bool) -> SketchPlacementRecipe {
    SketchPlacementRecipe::arc_slot([2., 3.], [[12., 3.], [2., 13.]], 2., ccw).unwrap()
}
fn distance(a: [f64; 2], b: [f64; 2]) -> f64 {
    (a[0] - b[0]).hypot(a[1] - b[1])
}
#[test]
fn preview_refuses_invalid_options_and_excessive_native_sampling() {
    let recipe = arc(true);
    for deflection in [0., -1., f64::NAN, f64::INFINITY, f64::MIN_POSITIVE] {
        assert!(sample_placement_preview(&recipe, options(deflection)).is_err());
    }
    for maximum_curve_points in [0, 3, u32::MAX] {
        let options = SketchSamplingOptionsV1 {
            deflection: 0.1,
            maximum_curve_points,
        };
        assert!(sample_placement_preview(&recipe, options).is_err());
    }
}
#[test]
fn preview_refuses_malformed_recipe_indices() {
    let mut recipe = arc(true);
    recipe.members[0].path = SketchPathMember::Line([0, 999]);
    assert!(
        sample_placement_preview(&recipe, options(0.1))
            .unwrap_err()
            .contains("999")
    );
}
