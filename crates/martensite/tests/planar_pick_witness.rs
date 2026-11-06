// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::planar_curve::{PlanarCurveDefinition, PlanarCurveFailureKind, PlanarGeometry};
fn adaptor() -> Geom2dAdaptorCurve {
    let definition = PlanarCurveDefinition {
        geometry: PlanarGeometry::Line {
            origin: [0.; 2],
            direction: [1., 0.],
        },
        bounds: Some([-1., 1.]),
        left_offset: 0.,
    };
    let curve = crate::planar_curve::construct::build_curve(&definition).unwrap();
    Geom2dAdaptorCurve::with_curve(&curve).unwrap()
}
#[test]
fn native_parameter_and_position_witnesses_cannot_escape_the_result_gates() {
    let curve = adaptor();
    let query = Point2::new(0., 1.);
    for (parameter, position, message) in [
        (2., [2., 0.], "pick parameter 2"),
        (0., [0., 0.1], "position residual 0.1"),
        (0.5, [0.5, 0.], "normal stationarity residual 0.5"),
    ] {
        let cause = checked_stationary(
            &curve,
            query,
            [-1., 1.],
            POnCurv2d::new(parameter, Point2::new(position[0], position[1])),
            PlanarPickOptionsV1::default(),
        )
        .unwrap_err();
        assert_eq!(cause.kind(), PlanarCurveFailureKind::InvalidResult);
        assert!(cause.to_string().contains(message), "{cause}");
    }
}
#[test]
fn boundary_witness_does_not_require_an_orthogonal_tangent() {
    let hit = checked_stationary(
        &adaptor(),
        Point2::new(2., 1.),
        [-1., 1.],
        POnCurv2d::new(1., Point2::new(1., 0.)),
        PlanarPickOptionsV1::default(),
    )
    .unwrap();
    assert_eq!(hit.origin, PlanarPickOrigin::Boundary);
    assert_eq!(hit.stationarity_residual, None);
}
#[test]
fn nonfinite_coordinates_and_distance_overflow_are_named_result_failures() {
    for (point, query) in [
        (Point2::new(f64::NAN, 0.), Point2::new(0., 0.)),
        (Point2::new(-f64::MAX, 0.), Point2::new(f64::MAX, 0.)),
    ] {
        let cause = witness(point, 0., query, PlanarPickOrigin::Boundary).unwrap_err();
        assert_eq!(cause.kind(), PlanarCurveFailureKind::InvalidResult);
        assert!(
            cause
                .to_string()
                .contains("expected finite native parameter")
        );
    }
    for parameter in [f64::NAN, f64::INFINITY, -2.] {
        assert!(require_span_parameter(parameter, [-1., 1.]).is_err());
    }
    for residual in [f64::NAN, f64::INFINITY, 1.] {
        assert!(require_residual(residual, 0.5, "test").is_err());
    }
}
