// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::planar_curve::{
    PlanarCurveDefinition, PlanarCurveFailureKind, PlanarGeometry, construct::build_curve,
};
use rustycad_tkgeomalgo::int_res2d::Transition;
fn axis() -> CurveHandle {
    build_curve(&PlanarCurveDefinition {
        geometry: PlanarGeometry::Line {
            origin: [0.0; 2],
            direction: [1.0, 0.0],
        },
        bounds: Some([0.0, 2.0]),
        left_offset: 0.0,
    })
    .unwrap()
}
fn native_point(parameter: f64, position: [f64; 2]) -> IntersectionPoint {
    IntersectionPoint::new(
        Point2::new(position[0], position[1]),
        parameter,
        parameter,
        Transition::default(),
        Transition::default(),
        false,
    )
}
#[test]
fn incomplete_intersection_is_not_an_empty_success() {
    let curve = axis();
    let failure = extract(&GInter::new(), [&curve, &curve], 1e-8).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::Incomplete);
    assert!(
        failure
            .to_string()
            .contains("expected a completed native result")
    );
}
#[test]
fn corrupt_parameters_positions_and_residuals_are_refused() {
    let curve = axis();
    let samples = [
        (f64::NAN, [0.0; 2]),
        (-1.0, [-1.0, 0.0]),
        (3.0, [3.0, 0.0]),
        (1.0, [f64::NAN, 0.0]),
        (1.0, [1.0, 1.0]),
    ];
    for (parameter, position) in samples {
        let failure =
            convert_point(native_point(parameter, position), [&curve, &curve], 1e-8).unwrap_err();
        assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidResult);
        assert!(failure.to_string().contains("expected"));
    }
    let failure = residual(&CurveHandle::default(), 0.0, Point2::default(), 1e-8).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidResult);
}
#[test]
fn supplied_residual_limit_is_enforced_without_a_hidden_floor() {
    let curve = axis();
    let hit = native_point(1.0, [1.0, 1e-12]);
    assert!(convert_point(hit, [&curve, &curve], 1e-13).is_err());
    let accepted = convert_point(hit, [&curve, &curve], 1e-12).unwrap();
    assert_eq!(accepted.residuals, [1e-12; 2]);
}
#[test]
fn half_bounded_overlap_preserves_the_missing_endpoint() {
    let curve = axis();
    for first in [false, true] {
        let segment =
            IntersectionSegment::half_bounded(native_point(1.0, [1.0, 0.0]), first, false, false);
        let overlap = convert_overlap(segment, [&curve, &curve], 1e-8).unwrap();
        assert_eq!(overlap.first.is_some(), first);
        assert_eq!(overlap.last.is_some(), !first);
    }
}
#[test]
fn report_order_uses_operand_parameters_and_explicit_unbounded_endpoints() {
    let curve = axis();
    let first = convert_point(native_point(0.0, [0.0; 2]), [&curve, &curve], 1e-8).unwrap();
    let last = convert_point(native_point(1.0, [1.0, 0.0]), [&curve, &curve], 1e-8).unwrap();
    let low = PlanarOverlap {
        first: None,
        last: Some(first.clone()),
        opposite: false,
    };
    let high = PlanarOverlap {
        first: Some(last.clone()),
        last: None,
        opposite: false,
    };
    let middle = PlanarOverlap {
        first: Some(first.clone()),
        last: Some(last.clone()),
        opposite: false,
    };
    let mut overlaps = vec![high.clone(), middle.clone(), low.clone()];
    sort_overlaps(&mut overlaps);
    assert_eq!(overlaps, [low, middle, high]);
    assert_eq!(compare_bounds(&None, &None), Ordering::Equal);
    assert_eq!(compare_bounds(&None, &Some(first.clone())), Ordering::Less);
    assert_eq!(compare_bounds(&Some(first), &None), Ordering::Greater);
    assert_eq!(compare_points(&last, &last), Ordering::Equal);
}
