// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{PlanarSplineDefinition, planar_curve::*};
fn line(bounds: Option<[f64; 2]>) -> PlanarCurveDefinition {
    PlanarCurveDefinition {
        geometry: PlanarGeometry::Line {
            origin: [0.; 2],
            direction: [1., 0.],
        },
        bounds,
        left_offset: 0.,
    }
}
fn pick(curve: &PlanarCurveDefinition, query: [f64; 2]) -> PlanarPickPoint {
    nearest_planar_curve_candidate(curve, query, PlanarPickOptionsV1::default()).unwrap()
}
#[test]
fn finite_segments_include_endpoints_when_orthogonal_projection_is_outside() {
    let segment = line(Some([-1., 1.]));
    let hit = pick(&segment, [2., 0.]);
    assert_eq!(
        (hit.parameter, hit.position, hit.distance),
        (1., [1., 0.], 1.)
    );
    assert_eq!(hit.origin, PlanarPickOrigin::Boundary);
    let interior = pick(&segment, [0., 3.]);
    assert_eq!((interior.position, interior.distance), ([0.; 2], 3.));
    assert!(
        interior.stationarity_residual.unwrap()
            <= PlanarPickOptionsV1::default().maximum_stationarity_residual
    );
}
#[test]
fn unbounded_and_offset_lines_share_the_same_query_path() {
    let mut curve = line(None);
    assert_eq!(pick(&curve, [2., 3.]).position, [2., 0.]);
    curve.left_offset = 0.25;
    assert_eq!(pick(&curve, [2., 3.]).position, [2., 0.25]);
}
#[test]
fn trimmed_arc_uses_its_nearer_end_instead_of_a_stationary_maximum() {
    let arc = PlanarCurveDefinition {
        geometry: PlanarGeometry::Circle {
            frame: PlanarConicFrame {
                center: [0.; 2],
                axis: [1., 0.],
                counterclockwise: true,
            },
            radius: 1.,
        },
        bounds: Some([0., std::f64::consts::FRAC_PI_2]),
        left_offset: 0.,
    };
    let hit = pick(&arc, [-2., 0.]);
    assert_eq!(hit.parameter, std::f64::consts::FRAC_PI_2);
    assert!((hit.distance - 5_f64.sqrt()).abs() <= 64. * f64::EPSILON);
    assert_eq!(hit.origin, PlanarPickOrigin::Boundary);
}
#[test]
fn c0_spline_internal_corner_is_a_domain_boundary_candidate() {
    let curve = PlanarCurveDefinition {
        geometry: PlanarGeometry::Spline(PlanarSplineDefinition {
            poles: vec![[-1., 0.], [0., 1.], [1., 0.]],
            weights: None,
            knots: vec![0., 1., 2.],
            multiplicities: vec![2, 1, 2],
            degree: 1,
            periodic: false,
        }),
        bounds: None,
        left_offset: 0.,
    };
    let hit = pick(&curve, [0., 1.25]);
    assert_eq!(
        (hit.parameter, hit.position, hit.distance),
        (1., [0., 1.], 0.25)
    );
    assert_eq!(hit.origin, PlanarPickOrigin::Boundary);
}
#[test]
fn invalid_queries_and_options_decline_before_native_construction() {
    let curve = line(Some([-1., 1.]));
    assert_eq!(
        nearest_planar_curve_candidate(&curve, [f64::NAN, 0.], PlanarPickOptionsV1::default())
            .unwrap_err()
            .kind(),
        PlanarCurveFailureKind::InvalidInput
    );
    let options = PlanarPickOptionsV1 {
        maximum_intervals: 0,
        ..Default::default()
    };
    assert_eq!(
        nearest_planar_curve_candidate(&curve, [0.; 2], options)
            .unwrap_err()
            .kind(),
        PlanarCurveFailureKind::InvalidInput
    );
}

#[test]
fn rational_quarter_circle_returns_a_stationary_radial_witness() {
    let curve = PlanarCurveDefinition {
        geometry: PlanarGeometry::Spline(PlanarSplineDefinition {
            poles: vec![[1., 0.], [1., 1.], [0., 1.]],
            weights: Some(vec![1., std::f64::consts::FRAC_1_SQRT_2, 1.]),
            knots: vec![0., 1.],
            multiplicities: vec![3, 3],
            degree: 2,
            periodic: false,
        }),
        bounds: None,
        left_offset: 0.,
    };
    let hit = pick(&curve, [2., 2.]);
    for coordinate in hit.position {
        assert!((coordinate - std::f64::consts::FRAC_1_SQRT_2).abs() < 1e-12);
    }
    assert!((hit.distance - (8_f64.sqrt() - 1.)).abs() < 1e-12);
    assert_eq!(hit.origin, PlanarPickOrigin::Stationary);
}
#[test]
fn ellipse_and_periodic_circle_have_repeatable_analytic_minima() {
    let frame = PlanarConicFrame {
        center: [0.; 2],
        axis: [1., 0.],
        counterclockwise: true,
    };
    for geometry in [
        PlanarGeometry::Ellipse {
            frame,
            radii: [2., 1.],
        },
        PlanarGeometry::Circle { frame, radius: 2. },
    ] {
        let curve = PlanarCurveDefinition {
            geometry,
            bounds: None,
            left_offset: 0.,
        };
        let expected = pick(&curve, [3., 0.]);
        assert!((expected.distance - 1.).abs() < 1e-12);
        assert!((expected.position[0] - 2.).abs() < 1e-12);
        assert!(expected.position[1].abs() < 1e-12);
        for _ in 0..16 {
            assert_eq!(pick(&curve, [3., 0.]), expected);
        }
    }
}

#[test]
fn model_entity_witness_keeps_identity_for_curves_and_hidden_defining_points() {
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId as Id};
    use sketch::nearest_sketch_entity_candidate as nearest;
    let mut sketch = Sketch2D::default();
    sketch.add_defining_point(Id(1), [0., 0.]).unwrap();
    sketch.add_defining_point(Id(2), [2., 0.]).unwrap();
    sketch.add_line(Id(3), [Id(1), Id(2)]).unwrap();
    let before = sketch.clone();
    let options = PlanarPickOptionsV1::default();
    let endpoint = nearest(&sketch, Id(2), [3., 0.], options).unwrap();
    assert_eq!(
        (endpoint.entity, endpoint.position, endpoint.distance),
        (Id(2), [2., 0.], 1.)
    );
    assert!(endpoint.curve.is_none());
    let segment = nearest(&sketch, Id(3), [3., 0.], options).unwrap();
    assert_eq!(
        (segment.entity, segment.position, segment.distance),
        (Id(3), [2., 0.], 1.)
    );
    assert!(segment.curve.is_some());
    assert!(sketch.same_recipe(&before));
    assert_eq!(
        nearest(&sketch, Id(99), [0.; 2], options)
            .unwrap_err()
            .entity,
        Id(99)
    );
    assert_eq!(
        nearest(&sketch, Id(2), [f64::NAN, 0.], options)
            .unwrap_err()
            .entity,
        Id(2)
    );
}

#[test]
fn trimmed_spline_queries_respect_authored_bounds_and_interval_budget() {
    let mut curve = PlanarCurveDefinition {
        geometry: PlanarGeometry::Spline(PlanarSplineDefinition {
            poles: vec![[-1., 0.], [0., 1.], [1., 0.]],
            weights: None,
            knots: vec![0., 1., 2.],
            multiplicities: vec![2, 1, 2],
            degree: 1,
            periodic: false,
        }),
        bounds: None,
        left_offset: 0.,
    };
    let options = PlanarPickOptionsV1 {
        maximum_intervals: 1,
        ..Default::default()
    };
    let cause = nearest_planar_curve_candidate(&curve, [0., 1.25], options).unwrap_err();
    assert!(
        cause.to_string().contains("pick interval count 2"),
        "{cause}"
    );
    curve.bounds = Some([0.25, 0.75]);
    let hit = nearest_planar_curve_candidate(&curve, [0., 1.25], options).unwrap();
    assert_eq!((hit.parameter, hit.position), (0.75, [-0.25, 0.75]));
    assert_eq!(hit.origin, PlanarPickOrigin::Boundary);
}

#[test]
fn invalid_native_pick_limits_decline_with_the_offending_policy() {
    let curve = line(Some([-1., 1.]));
    for options in [
        PlanarPickOptionsV1 {
            native_tolerance: f64::NAN,
            ..Default::default()
        },
        PlanarPickOptionsV1 {
            maximum_position_residual: 0.,
            ..Default::default()
        },
        PlanarPickOptionsV1 {
            maximum_stationarity_residual: -1.,
            ..Default::default()
        },
        PlanarPickOptionsV1 {
            maximum_intervals: i32::MAX as usize,
            ..Default::default()
        },
    ] {
        let cause = nearest_planar_curve_candidate(&curve, [0.; 2], options).unwrap_err();
        assert_eq!(cause.kind(), PlanarCurveFailureKind::InvalidInput);
        assert!(cause.to_string().contains(&format!("{options:?}")));
    }
}
