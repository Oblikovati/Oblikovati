// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::{
    CircumcircleOptions, PlanarCurveFailureKind, construct_planar_circumcircle,
};
fn options(bound: f64) -> CircumcircleOptions {
    CircumcircleOptions {
        maximum_point_residual: bound,
    }
}
#[test]
fn exact_cardinal_circle_and_both_input_orientations_preserve_owned_results() {
    let first = [[2.0, 0.0], [0.0, 2.0], [-2.0, 0.0]];
    let reverse = [first[2], first[1], first[0]];
    let a = construct_planar_circumcircle(first, options(0.0)).unwrap();
    let b = construct_planar_circumcircle(reverse, options(0.0)).unwrap();
    assert_eq!(a.center, [0.0; 2]);
    assert_eq!(a.radius, 2.0);
    assert_eq!(a.point_residuals, [0.0; 3]);
    assert!(a.counter_clockwise);
    assert_eq!(b.center, a.center);
    assert_eq!(b.radius, a.radius);
    assert!(!b.counter_clockwise);
    assert_eq!(
        construct_planar_circumcircle(first, options(0.0)).unwrap(),
        a
    );
}
#[test]
fn exact_collinearity_and_coincident_points_decline_before_native_construction() {
    for points in [
        [[0.0; 2]; 3],
        [[0.0; 2], [1.0, 1.0], [2.0, 2.0]],
        [[1e100, 1e100], [2e100, 2e100], [3e100, 3e100]],
    ] {
        let error = construct_planar_circumcircle(points, options(1.0)).unwrap_err();
        assert_eq!(error.kind(), PlanarCurveFailureKind::InvalidInput);
        assert!(error.to_string().contains("noncollinear"));
    }
}
#[test]
fn invalid_bounds_and_each_nonfinite_input_axis_have_typed_context() {
    let points = [[2.0, 0.0], [0.0, 2.0], [-2.0, 0.0]];
    for bound in [-1.0, f64::NAN, f64::INFINITY] {
        let error = construct_planar_circumcircle(points, options(bound)).unwrap_err();
        assert_eq!(error.kind(), PlanarCurveFailureKind::InvalidInput);
        assert!(error.to_string().contains("bound"));
    }
    for index in 0..3 {
        for axis in 0..2 {
            for bad in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
                let mut input = points;
                input[index][axis] = bad;
                let error = construct_planar_circumcircle(input, options(1.0)).unwrap_err();
                assert_eq!(error.kind(), PlanarCurveFailureKind::InvalidInput);
                assert!(
                    error
                        .to_string()
                        .contains(&format!("point {index} axis {axis}"))
                );
            }
        }
    }
}
#[test]
fn native_parallel_sentinel_and_collapsed_radius_are_never_successful_circles() {
    for points in [
        [[0.0; 2], [1.0, 0.0], [2.0, 1e-20]],
        [[0.0; 2], [1e-200, 0.0], [0.0, 1e-200]],
    ] {
        let error = construct_planar_circumcircle(points, options(f64::MAX)).unwrap_err();
        assert_eq!(error.kind(), PlanarCurveFailureKind::InvalidResult);
        assert!(error.to_string().contains("native circle"));
    }
}
#[derive(serde::Deserialize)]
struct NativeCapture {
    version: String,
    cases: Vec<NativeCircle>,
}
#[derive(serde::Deserialize)]
struct NativeCircle {
    points: [[f64; 2]; 3],
    center: [f64; 2],
    radius: f64,
}
#[test]
fn released_builder_matches_independent_native_occt_constructions() {
    let capture: NativeCapture =
        serde_json::from_str(include_str!("circumcircle/occt-circles.json")).unwrap();
    assert!(!capture.version.is_empty());
    assert_eq!(capture.cases.len(), 4);
    for case in capture.cases {
        let result = construct_planar_circumcircle(case.points, options(1e-7)).unwrap();
        for (actual, expected) in result
            .center
            .into_iter()
            .chain([result.radius])
            .zip(case.center.into_iter().chain([case.radius]))
        {
            let bound = 64.0 * f64::EPSILON * expected.abs().max(1.0);
            assert!(
                (actual - expected).abs() <= bound,
                "points {:?}: actual {actual}, expected {expected}, bound {bound}",
                case.points
            );
        }
    }
}
#[test]
fn residual_acceptance_is_explicit_and_a_failed_call_cannot_replace_a_prior_result() {
    let points = [[0.1, 0.2], [2.3, -1.7], [4.4, 5.6]];
    let accepted = construct_planar_circumcircle(points, options(1e-12)).unwrap();
    assert!(accepted.point_residuals.iter().any(|r| *r > 0.0));
    let rejected = construct_planar_circumcircle(points, options(0.0)).unwrap_err();
    assert_eq!(rejected.kind(), PlanarCurveFailureKind::InvalidResult);
    assert!(rejected.to_string().contains("residuals"));
    assert_eq!(
        construct_planar_circumcircle(points, options(1e-12)).unwrap(),
        accepted
    );
}
#[test]
fn finite_extreme_input_failure_does_not_publish_invalid_geometry() {
    for points in [
        [[f64::MAX, 0.0], [-f64::MAX, 0.0], [0.0, f64::MAX]],
        [[0.0; 2], [1e-300, 0.0], [0.0, 1e-300]],
    ] {
        let error = construct_planar_circumcircle(points, options(1.0)).unwrap_err();
        assert!(matches!(
            error.kind(),
            PlanarCurveFailureKind::Kernel | PlanarCurveFailureKind::InvalidResult
        ));
        assert!(error.to_string().contains("circumcircle points"));
    }
}
#[test]
fn generated_translated_circles_preserve_all_point_permutations_across_scales() {
    for exponent in [-100, -40, 0, 40, 100] {
        let scale = 2.0_f64.powi(exponent);
        let center = [3.0 * scale, -7.0 * scale];
        let points = [
            [5.0 * scale, -7.0 * scale],
            [3.0 * scale, -5.0 * scale],
            [scale, -7.0 * scale],
        ];
        for (indices, ccw) in [
            ([0, 1, 2], true),
            ([1, 2, 0], true),
            ([2, 0, 1], true),
            ([2, 1, 0], false),
            ([1, 0, 2], false),
            ([0, 2, 1], false),
        ] {
            let ordered = indices.map(|i| points[i]);
            let bound = 64.0 * f64::EPSILON * scale;
            let report = construct_planar_circumcircle(ordered, options(bound)).unwrap();
            for (actual, expected) in report.center.into_iter().zip(center) {
                assert!((actual - expected).abs() <= bound);
            }
            assert!((report.radius - 2.0 * scale).abs() <= bound);
            assert_eq!(report.counter_clockwise, ccw);
            assert!(report.point_residuals.iter().all(|r| *r <= bound));
        }
    }
}
#[test]
fn unrepresentable_tiny_center_retains_the_native_exception_as_its_source() {
    use std::error::Error;
    let next = f64::from_bits(1.0_f64.to_bits() + 1);
    let points = [[1.0, 1.0], [next, 1.0], [1.0, next]];
    let error = construct_planar_circumcircle(points, options(1e-12)).unwrap_err();
    assert_eq!(error.kind(), PlanarCurveFailureKind::Kernel);
    assert!(
        error
            .source()
            .unwrap()
            .downcast_ref::<martensite_tkernel::standard::Failure>()
            .is_some()
    );
    assert!(error.to_string().contains("zero norm"));
}
#[test]
fn injected_constructor_and_final_verifier_keep_the_same_owned_geometry_contract() {
    use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    use oblikovati_model::sketch::{SketchCircleConstruction, SolvedCircularSeed};
    let backend = MartensiteSketchConstruction;
    let points = [[2.0, 0.0], [0.0, 2.0], [-2.0, 0.0]];
    let circle = backend.circumcircle(points, options(0.0)).unwrap();
    let solved = SolvedCircularSeed {
        center: circle.center,
        radius: circle.radius,
        arc: Some(([points[0], points[2]], true)),
    };
    backend.verify(points, solved, options(0.0)).unwrap();
    backend
        .verify(
            points,
            SolvedCircularSeed {
                arc: None,
                ..solved
            },
            options(0.0),
        )
        .unwrap();
    for bad in [
        SolvedCircularSeed {
            radius: 0.0,
            ..solved
        },
        SolvedCircularSeed {
            radius: f64::NAN,
            ..solved
        },
        SolvedCircularSeed {
            center: [f64::INFINITY, 0.0],
            ..solved
        },
        SolvedCircularSeed {
            radius: 3.0,
            ..solved
        },
        SolvedCircularSeed {
            arc: Some(([points[0], points[2]], false)),
            ..solved
        },
        SolvedCircularSeed {
            arc: Some(([[2.0, 0.1], points[2]], true)),
            ..solved
        },
        SolvedCircularSeed {
            arc: Some(([[f64::NAN, 0.0], points[2]], true)),
            ..solved
        },
        SolvedCircularSeed {
            arc: Some(([[0.0, -2.0], points[2]], true)),
            ..solved
        },
    ] {
        assert!(backend.verify(points, bad, options(0.0)).is_err());
    }
    let error = backend
        .circumcircle([[0.0; 2]; 3], options(0.0))
        .unwrap_err();
    assert!(
        error
            .downcast_ref::<oblikovati_martensite::planar_curve::PlanarCurveFailure>()
            .is_some()
    );
}

#[test]
fn centered_arc_winding_uses_exact_orientation_and_accepts_semicircles() {
    use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    use oblikovati_model::sketch::SketchCircleConstruction;
    let backend = MartensiteSketchConstruction;
    for scale in [1e-200, 1., 1e200] {
        let center = [0., 0.];
        let start = [scale, 0.];
        assert!(backend.arc_direction([center, start, [0., scale]]).unwrap());
        assert!(
            !backend
                .arc_direction([center, start, [0., -scale]])
                .unwrap()
        );
        assert!(
            !backend
                .arc_direction([center, start, [-scale, 0.]])
                .unwrap()
        );
        assert!(
            !backend
                .arc_direction([center, [0., scale], [0., -scale]])
                .unwrap()
        );
    }
}
#[test]
fn centered_arc_refuses_nonfinite_zero_and_identical_radial_directions() {
    use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    use oblikovati_model::sketch::SketchCircleConstruction;
    for points in [
        [[0., 0.], [1., 0.], [2., 0.]],
        [[0., 0.], [0., 1.], [0., 2.]],
        [[0., 0.], [0., 0.], [1., 0.]],
        [[0., 0.], [1., 0.], [0., 0.]],
        [[f64::NAN, 0.], [1., 0.], [0., 1.]],
    ] {
        assert!(MartensiteSketchConstruction.arc_direction(points).is_err());
    }
}
