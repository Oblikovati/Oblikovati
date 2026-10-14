// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    PlacementPreviewCurve, SketchSamplingOptionsV1, sample_placement_preview,
};
use oblikovati_model::sketch::{SketchPathMember, SketchPlacementGeometry, SketchPlacementRecipe};
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
    let SketchPlacementGeometry::Path(path) = member.geometry else {
        panic!("expected path")
    };
    let endpoints = match path {
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
    } = path
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
    recipe.members[0].geometry = SketchPlacementGeometry::Path(SketchPathMember::Line([0, 999]));
    assert!(
        sample_placement_preview(&recipe, options(0.1))
            .unwrap_err()
            .contains("999")
    );
}

#[test]
fn circle_preview_is_closed_and_matches_the_canonical_circle_without_mutation() {
    let recipe = SketchPlacementRecipe::circle([3., -4.], 5.).unwrap();
    let original = recipe.clone();
    let curves = sample_placement_preview(&recipe, options(0.01)).unwrap();
    assert_eq!(recipe, original);
    assert_eq!(curves.len(), 1);
    assert_eq!(curves[0].member_index, 0);
    assert!(!curves[0].construction);
    let samples = &curves[0].points;
    assert!(distance(*samples.first().unwrap(), *samples.last().unwrap()) < 1e-12);
    for sample in samples {
        assert!((distance(*sample, [3., -4.]) - 5.).abs() < 1e-12);
    }
    verify_arc(&curves[0], [3., -4.], 5., true);
}

#[test]
fn unequal_arc_radii_are_solved_before_sampling_without_changing_the_recipe() {
    let recipe = SketchPlacementRecipe::arc([0., 0.], [[5., 0.], [0., 2.]], true).unwrap();
    let before = recipe.clone();
    let curves = sample_placement_preview(&recipe, options(0.01)).unwrap();
    assert_eq!(recipe, before);
    assert_eq!(curves.len(), 1);
    assert!(curves[0].points.len() > 2);
    assert_ne!(curves[0].points[0], recipe.points[1]);
}
#[test]
fn preview_refuses_failed_solve_and_circular_postconditions() {
    use oblikovati_model::sketch::{CircumcircleOptions, SketchCircularPlacementCheck};
    let mut circle = SketchPlacementRecipe::circle([0., 0.], 2.).unwrap();
    circle.circular_checks.push(SketchCircularPlacementCheck {
        member: 0,
        points: [[5., 0.], [0., 5.], [-5., 0.]],
        options: CircumcircleOptions {
            maximum_point_residual: 0.,
        },
    });
    assert!(
        sample_placement_preview(&circle, options(0.01))
            .unwrap_err()
            .contains("verification")
    );
    let mut arc = SketchPlacementRecipe::arc([0., 0.], [[5., 0.], [0., 2.]], true).unwrap();
    arc.points[0] = [-f64::MAX, 0.];
    arc.points[1] = [f64::MAX, 0.];
    assert!(
        sample_placement_preview(&arc, options(0.01))
            .unwrap_err()
            .contains("preview solve")
    );
}

#[test]
fn preview_respects_the_owner_iteration_limit_and_can_recover_with_its_normal_policy() {
    use oblikovati_martensite::planar_curve::sketch::sample_placement_preview_with_policy;
    use oblikovati_solve::nonlinear::SolveOptions;
    let recipe = SketchPlacementRecipe::arc([0., 0.], [[5., 0.], [0., 2.]], true).unwrap();
    let limited = SolveOptions {
        max_iterations: 1,
        ..Default::default()
    };
    let error = sample_placement_preview_with_policy(&recipe, options(0.01), limited).unwrap_err();
    assert!(error.contains("IterationLimit"), "{error}");
    assert!(
        sample_placement_preview_with_policy(&recipe, options(0.01), Default::default()).is_ok()
    );
}
