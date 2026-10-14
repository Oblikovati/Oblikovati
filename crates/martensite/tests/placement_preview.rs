// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    PlacementPreviewMember, SketchDisplayPrimitive, SketchSamplingOptionsV1,
    sample_placement_preview,
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
        fine.iter().map(|c| curve_points(c).len()).sum::<usize>()
            > curves.iter().map(|c| curve_points(c).len()).sum()
    );
}
fn verify_member(recipe: &SketchPlacementRecipe, index: usize, curve: &PlacementPreviewMember) {
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
    for (sample, id) in [
        curve_points(curve).first().unwrap(),
        curve_points(curve).last().unwrap(),
    ]
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
        assert_eq!(curve_points(curve).len(), 2);
    }
}
fn verify_arc(curve: &PlacementPreviewMember, center: [f64; 2], radius: f64, ccw: bool) {
    for pair in curve_points(curve).windows(2) {
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
    let samples = curve_points(&curves[0]);
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
    assert!(curve_points(&curves[0]).len() > 2);
    assert_ne!(curve_points(&curves[0])[0], recipe.points[1]);
}
#[test]
fn preview_refuses_failed_solve_and_circular_postconditions() {
    use oblikovati_model::sketch::{CircumcircleOptions, SketchCircularPlacementCheck};
    let mut circle = SketchPlacementRecipe::circle([0., 0.], 2.).unwrap();
    circle.circular_checks.push(SketchCircularPlacementCheck {
        member: 0,
        cap_members: None,
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

#[test]
fn ellipse_preview_is_closed_and_uses_the_native_analytic_frame() {
    let recipe = SketchPlacementRecipe::ellipse([3., 4.], [0.6, 0.8], [5., 2.]).unwrap();
    let before = recipe.clone();
    let curves = sample_placement_preview(&recipe, options(0.01)).unwrap();
    assert_eq!(recipe, before);
    assert_eq!(curves.len(), 1);
    let curve = &curves[0];
    assert_eq!(curve.member_index, 0);
    assert!(!curve.construction);
    assert!(
        distance(
            *curve_points(curve).first().unwrap(),
            *curve_points(curve).last().unwrap()
        ) < 1e-12
    );
    for point in curve_points(curve) {
        let x = point[0] - 3.;
        let y = point[1] - 4.;
        let major = (0.6 * x + 0.8 * y) / 5.;
        let minor = (-0.8 * x + 0.6 * y) / 2.;
        assert!((major * major + minor * minor - 1.).abs() < 1e-12);
    }
    let fine = sample_placement_preview(&recipe, options(0.001)).unwrap();
    assert!(curve_points(&fine[0]).len() > curve_points(curve).len());
}

#[test]
fn off_radius_arc_slot_cursor_keeps_four_nondegenerate_boundary_arcs() {
    let recipe =
        SketchPlacementRecipe::arc_slot([0., 0.], [[10., 0.], [5., 5.]], 2., true).unwrap();
    let curves = sample_placement_preview(&recipe, options(0.01)).unwrap();
    assert_eq!(curves.len(), 4);
    for curve in &curves {
        assert!(curve_points(curve).len() > 2);
        assert!(
            distance(
                *curve_points(curve).first().unwrap(),
                *curve_points(curve).last().unwrap()
            ) > 0.1
        );
    }
}

#[test]
fn three_point_slot_preview_checks_solved_centerline_and_rejects_wrong_caps() {
    use oblikovati_model::sketch::{CircumcircleOptions, SketchCircularPlacementCheck};
    let mut recipe =
        SketchPlacementRecipe::arc_slot([0., 0.], [[5., 0.], [0., 5.]], 2., true).unwrap();
    recipe.circular_checks.push(SketchCircularPlacementCheck {
        member: 0,
        cap_members: Some([3, 1]),
        points: [[5., 0.], [3., 4.], [0., 5.]],
        options: CircumcircleOptions {
            maximum_point_residual: 1e-9,
        },
    });
    assert_eq!(
        sample_placement_preview(&recipe, options(0.01))
            .unwrap()
            .len(),
        4
    );
    recipe.circular_checks[0].cap_members = Some([1, 3]);
    assert!(
        sample_placement_preview(&recipe, options(0.01))
            .unwrap_err()
            .contains("displacement")
    );
    recipe.circular_checks[0].cap_members = Some([3, 99]);
    assert!(sample_placement_preview(&recipe, options(0.01)).is_err());
}

fn curve_points(member: &PlacementPreviewMember) -> &[[f64; 2]] {
    let SketchDisplayPrimitive::Curve(points) = &member.primitive else {
        panic!("expected curve samples, got {:?}", member.primitive)
    };
    points
}

#[test]
fn point_preview_retains_position_and_is_a_marker_instead_of_a_curve() {
    for position in [[0., 0.], [3., -4.], [-20., 30.], [0.125, 100.]] {
        let recipe = SketchPlacementRecipe::point(position).unwrap();
        let before = recipe.clone();
        let sampled = sample_placement_preview(&recipe, options(0.01)).unwrap();
        assert_eq!(recipe, before);
        assert_eq!(
            sampled,
            vec![PlacementPreviewMember {
                member_index: 0,
                construction: false,
                primitive: SketchDisplayPrimitive::Point(position),
            }]
        );
        assert_eq!(
            sampled,
            sample_placement_preview(&recipe, options(0.001)).unwrap()
        );
    }
}

#[test]
fn point_and_curve_members_keep_shared_positions_and_authored_order() {
    use oblikovati_model::sketch::SketchPlacementMember;
    let mut recipe = SketchPlacementRecipe::point([3., -4.]).unwrap();
    recipe.points.push([8., -4.]);
    recipe.members.push(SketchPlacementMember {
        geometry: SketchPlacementGeometry::Path(SketchPathMember::Line([0, 1])),
        construction: true,
    });
    let sampled = sample_placement_preview(&recipe, options(0.01)).unwrap();
    assert_eq!(sampled.len(), 2);
    assert_eq!(
        sampled[0].primitive,
        SketchDisplayPrimitive::Point([3., -4.])
    );
    assert_eq!(sampled[1].member_index, 1);
    assert!(sampled[1].construction);
    assert_eq!(
        sampled[1].primitive,
        SketchDisplayPrimitive::Curve(recipe.points)
    );
}
