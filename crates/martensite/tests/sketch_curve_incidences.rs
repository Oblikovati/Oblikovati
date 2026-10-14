// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::{
    PlanarCurveFailureKind, PlanarGeometry, PlanarIntersectionOptions, sketch::*,
};
use oblikovati_model::sketch::{
    Sketch2D, SketchEllipse2D, SketchEntityId, SketchSpline2D, SketchSplineKind,
};
use std::error::Error;
fn options() -> SketchIntersectionOptionsV1 {
    SketchIntersectionOptionsV1 {
        intersection: PlanarIntersectionOptions {
            confusion_tolerance: 1e-9,
            intersection_tolerance: 1e-10,
            maximum_point_residual: 1e-8,
        },
        maximum_endpoint_residual: 1e-8,
    }
}
fn add_line(sketch: &mut Sketch2D, id: u64, ends: [[f64; 2]; 2]) {
    sketch.add_point(SketchEntityId(id + 1), ends[0]).unwrap();
    sketch.add_point(SketchEntityId(id + 2), ends[1]).unwrap();
    sketch
        .add_line(
            SketchEntityId(id),
            [SketchEntityId(id + 1), SketchEntityId(id + 2)],
        )
        .unwrap();
}
fn source() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 10, [[0.0, 0.0], [1.0, 0.0]]);
    sketch
}
fn source_with_control_spline() -> Sketch2D {
    let mut sketch = source();
    sketch
        .add_defining_point(SketchEntityId(41), [0.5, -1.])
        .unwrap();
    sketch
        .add_defining_point(SketchEntityId(42), [0.5, 1.])
        .unwrap();
    sketch
        .add_spline(
            SketchSpline2D::new(
                SketchEntityId(40),
                vec![SketchEntityId(41), SketchEntityId(42)],
                false,
                SketchSplineKind::Control,
            )
            .unwrap(),
        )
        .unwrap();
    sketch
}
#[test]
fn constructed_control_spline_participates_in_analytic_line_intersections() {
    let report = intersect_sketch_line(
        &source_with_control_spline(),
        SketchEntityId(10),
        SketchLineDomain::Segment,
        options(),
    )
    .unwrap();
    assert_eq!(report.targets.len(), 1);
    assert_eq!(report.targets[0].target.entity, SketchEntityId(40));
    assert_eq!(report.targets[0].target.endpoint_residuals, Some([0.; 2]));
    assert_eq!(report.targets[0].intersection.points.len(), 1);
    assert_eq!(report.targets[0].intersection.points[0].position, [0.5, 0.]);
}
#[test]
fn finite_segments_and_infinite_supports_preserve_the_go_line_domain_distinction() {
    let mut sketch = source();
    add_line(&mut sketch, 20, [[3.0, 2.0], [3.0, 3.0]]);
    let before = sketch.clone();
    let finite = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Segment,
        options(),
    )
    .unwrap();
    assert!(finite.targets[0].intersection.points.is_empty());
    let support = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap();
    assert_eq!(support.targets[0].intersection.points.len(), 1);
    assert_eq!(
        support.targets[0].intersection.points[0].position,
        [3.0, 0.0]
    );
    assert_eq!(
        support.targets[0].intersection.points[0].parameters,
        [3.0, -2.0]
    );
    assert_eq!(support.source.endpoint_residuals, Some([0.0; 2]));
    assert_eq!(support.line_domain, SketchLineDomain::Support);
    assert_eq!(support.options, options());
    assert_eq!(before, sketch);
}
#[test]
fn full_conics_keep_axes_radii_identity_and_all_roots_in_numeric_identity_order() {
    let mut sketch = source();
    sketch.add_point(SketchEntityId(91), [9.0, 0.0]).unwrap();
    sketch
        .add_ellipse(
            SketchEllipse2D::new(
                SketchEntityId(90),
                SketchEntityId(91),
                [0.0, 1.0],
                [2.0, 1.0],
            )
            .unwrap(),
        )
        .unwrap();
    sketch.add_point(SketchEntityId(31), [5.0, 0.0]).unwrap();
    sketch
        .add_circle(SketchEntityId(30), SketchEntityId(31), 1.0)
        .unwrap();
    let report = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap();
    assert_eq!(
        report
            .targets
            .iter()
            .map(|row| row.target.entity.0)
            .collect::<Vec<_>>(),
        [30, 90]
    );
    let xs: Vec<f64> = report
        .targets
        .iter()
        .flat_map(|row| &row.intersection.points)
        .map(|point| point.position[0])
        .collect();
    assert_eq!(xs.len(), 4);
    for (&actual, expected) in xs.iter().zip([4.0, 6.0, 8.0, 10.0]) {
        assert!((actual - expected).abs() < 1e-10);
    }
    assert!(
        report
            .targets
            .iter()
            .all(|row| row.target.endpoint_residuals.is_none())
    );
    assert!(matches!(
        report.targets[1].target.definition.geometry,
        PlanarGeometry::Ellipse {
            radii: [2.0, 1.0],
            ..
        }
    ));
}
fn arc_fixture(counterclockwise: bool, end: [f64; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 10, [[0.0, 1.0], [1.0, 1.0]]);
    for (id, position) in [(21, [5.0, 0.0]), (22, [7.0, 0.0]), (23, end)] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    sketch
        .add_arc(
            SketchEntityId(20),
            SketchEntityId(21),
            [SketchEntityId(22), SketchEntityId(23)],
            counterclockwise,
        )
        .unwrap();
    sketch
}
#[test]
fn directed_arcs_keep_the_authored_sweep_including_the_complementary_clockwise_arc() {
    for counterclockwise in [false, true] {
        let sketch = arc_fixture(counterclockwise, [5.0, 2.0]);
        let report = intersect_sketch_line(
            &sketch,
            SketchEntityId(10),
            SketchLineDomain::Support,
            options(),
        )
        .unwrap();
        let target = &report.targets[0];
        assert_eq!(target.intersection.points.len(), 1);
        let sign = if counterclockwise { 1.0 } else { -1.0 };
        assert!(
            (target.intersection.points[0].position[0] - (5.0 + sign * 3.0_f64.sqrt())).abs()
                < 1e-10
        );
        assert_eq!(target.target.definition.bounds.unwrap()[0], 0.0);
        assert!(
            target
                .target
                .endpoint_residuals
                .unwrap()
                .into_iter()
                .all(|v| v <= options().maximum_endpoint_residual)
        );
    }
}
#[test]
fn invalid_arc_endpoints_are_not_silently_projected_or_turned_into_full_circles() {
    for end in [[5.0, 3.0], [7.0, 0.0], [8.0, 0.0]] {
        let sketch = arc_fixture(true, end);
        let before = sketch.clone();
        let failure = intersect_sketch_line(
            &sketch,
            SketchEntityId(10),
            SketchLineDomain::Support,
            options(),
        )
        .unwrap_err();
        assert_eq!(failure.entity, SketchEntityId(20));
        assert!(failure.to_string().contains("expected"));
        assert!(failure.source().is_some());
        assert_eq!(before, sketch);
    }
}
#[test]
fn tangent_contacts_and_unbounded_overlaps_remain_distinct_results() {
    let mut sketch = source();
    add_line(&mut sketch, 20, [[4.0, 0.0], [2.0, 0.0]]);
    sketch.add_point(SketchEntityId(31), [5.0, 1.0]).unwrap();
    sketch
        .add_circle(SketchEntityId(30), SketchEntityId(31), 1.0)
        .unwrap();
    let report = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap();
    let overlap = &report.targets[0].intersection.overlaps[0];
    assert!(overlap.first.is_none() && overlap.last.is_none() && overlap.opposite);
    assert_eq!(report.targets[1].intersection.points.len(), 1);
    assert_eq!(
        report.targets[1].intersection.points[0].position,
        [5.0, 0.0]
    );
}
#[test]
fn invalid_source_and_options_fail_even_when_no_target_curves_exist() {
    let sketch = source();
    for selected in [SketchEntityId(999), SketchEntityId(11)] {
        let failure =
            intersect_sketch_line(&sketch, selected, SketchLineDomain::Support, options())
                .unwrap_err();
        assert!(matches!(failure.cause, SketchCurveFailureCause::Model(_)));
        assert!(failure.source().unwrap().to_string().contains("expected"));
    }
    for limit in [0.0, -1.0, f64::NAN, f64::INFINITY] {
        let policy = SketchIntersectionOptionsV1 {
            maximum_endpoint_residual: limit,
            ..options()
        };
        assert!(
            intersect_sketch_line(
                &sketch,
                SketchEntityId(10),
                SketchLineDomain::Support,
                policy
            )
            .is_err()
        );
    }
    let mut policy = options();
    policy.intersection.intersection_tolerance = 0.0;
    assert!(
        intersect_sketch_line(
            &sketch,
            SketchEntityId(10),
            SketchLineDomain::Support,
            policy
        )
        .is_err()
    );
}
#[test]
fn native_failures_keep_the_entity_and_original_exception_chain() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 10, [[0.0; 2]; 2]);
    let failure = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap_err();
    let SketchCurveFailureCause::Curve(ref cause) = failure.cause else {
        panic!("{failure:?}")
    };
    assert_eq!(cause.kind(), PlanarCurveFailureKind::Kernel);
    assert!(failure.source().unwrap().source().is_some());
    assert!(failure.to_string().contains("SketchEntityId(10)"));
}
#[test]
fn points_are_noncurves_but_construction_and_centerline_curves_still_participate() {
    let mut sketch = source();
    add_line(&mut sketch, 20, [[3.0, -2.0], [3.0, 2.0]]);
    sketch
        .set_curve_construction(&[SketchEntityId(20)], true)
        .unwrap();
    let report = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap();
    assert_eq!(report.targets.len(), 1);
    assert_eq!(report.targets[0].intersection.points.len(), 1);
}
#[test]
fn degenerate_target_is_not_hidden_by_a_released_spline() {
    let mut sketch = source();
    add_line(&mut sketch, 20, [[0.0; 2]; 2]);
    let spline = SketchSpline2D::new(
        SketchEntityId(40),
        vec![SketchEntityId(11), SketchEntityId(12)],
        false,
        SketchSplineKind::Control,
    )
    .unwrap();
    sketch.add_spline(spline).unwrap();
    let before = sketch.clone();
    let failure = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        options(),
    )
    .unwrap_err();
    assert_eq!(failure.entity, SketchEntityId(20));
    assert!(!failure.to_string().contains("issue #9"));
    let SketchCurveFailureCause::Curve(cause) = failure.cause else {
        panic!("expected typed curve failure")
    };
    assert_eq!(cause.kind(), PlanarCurveFailureKind::Kernel);
    assert_eq!(before, sketch);
}
#[test]
fn endpoint_contacts_and_empty_target_sets_do_not_disappear_from_the_report() {
    let mut sketch = source();
    let empty = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Segment,
        options(),
    )
    .unwrap();
    assert!(empty.targets.is_empty());
    add_line(&mut sketch, 20, [[1.0, 0.0], [1.0, 1.0]]);
    sketch
        .set_line_centerline(SketchEntityId(20), true)
        .unwrap();
    let report = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Segment,
        options(),
    )
    .unwrap();
    assert_eq!(
        report.targets[0].intersection.points[0].parameters,
        [1.0, 0.0]
    );
    assert_eq!(report.targets[0].target.definition.bounds, Some([0.0, 1.0]));
}
#[test]
fn query_parameters_use_working_length_and_results_own_their_source_values() {
    for scale in [1e-6, 1.0, 1e6] {
        let mut sketch = Sketch2D::default();
        add_line(&mut sketch, 10, [[0.0, 0.0], [scale, 0.0]]);
        add_line(
            &mut sketch,
            20,
            [[3.0 * scale, -scale], [3.0 * scale, scale]],
        );
        let report = intersect_sketch_line(
            &sketch,
            SketchEntityId(10),
            SketchLineDomain::Support,
            options(),
        )
        .unwrap();
        let point = &report.targets[0].intersection.points[0];
        assert!((point.parameters[0] - 3.0 * scale).abs() <= scale * 1e-12);
        let retained = report.clone();
        sketch
            .set_curve_construction(&[SketchEntityId(20)], true)
            .unwrap();
        drop(sketch);
        assert_eq!(report, retained);
    }
}
#[test]
fn strict_native_result_bounds_identify_the_target_without_partial_success() {
    let mut sketch = source();
    sketch.add_point(SketchEntityId(31), [2.3, 0.5]).unwrap();
    sketch
        .add_circle(SketchEntityId(30), SketchEntityId(31), 1.1)
        .unwrap();
    let mut policy = options();
    policy.intersection.maximum_point_residual = f64::MIN_POSITIVE;
    let failure = intersect_sketch_line(
        &sketch,
        SketchEntityId(10),
        SketchLineDomain::Support,
        policy,
    )
    .unwrap_err();
    assert_eq!(failure.entity, SketchEntityId(30));
    let SketchCurveFailureCause::Curve(cause) = failure.cause else {
        panic!("expected native result failure")
    };
    assert_eq!(cause.kind(), PlanarCurveFailureKind::InvalidResult);
}
