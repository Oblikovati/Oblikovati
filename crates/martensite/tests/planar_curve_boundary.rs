// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{PlanarSplineDefinition, planar_curve::*};
use std::error::Error;
fn options() -> PlanarIntersectionOptions {
    PlanarIntersectionOptions {
        confusion_tolerance: 1e-9,
        intersection_tolerance: 1e-9,
        maximum_point_residual: 1e-8,
    }
}
fn definition(geometry: PlanarGeometry) -> PlanarCurveDefinition {
    PlanarCurveDefinition {
        geometry,
        bounds: None,
        left_offset: 0.0,
    }
}
fn line(origin: [f64; 2], direction: [f64; 2]) -> PlanarCurveDefinition {
    definition(PlanarGeometry::Line { origin, direction })
}
fn circle(counterclockwise: bool) -> PlanarCurveDefinition {
    definition(PlanarGeometry::Circle {
        frame: conic_frame(counterclockwise),
        radius: 2.0,
    })
}
fn spline() -> PlanarCurveDefinition {
    definition(PlanarGeometry::Spline(PlanarSplineDefinition {
        poles: vec![[2.0, 0.0], [2.0, 2.0], [0.0, 2.0]],
        weights: Some(vec![1.0, std::f64::consts::FRAC_1_SQRT_2, 1.0]),
        knots: vec![0.0, 1.0],
        multiplicities: vec![3, 3],
        degree: 2,
        periodic: false,
    }))
}
#[test]
fn owned_inputs_and_reports_survive_independent_calls_and_failures() {
    let first = line([0.0; 2], [1.0, 0.0]);
    let mut second = line([1.0, -1.0], [0.0, 1.0]);
    let before = [first.clone(), second.clone()];
    let report = intersect_planar_curves(&first, &second, options()).unwrap();
    assert_eq!([first.clone(), second.clone()], before);
    assert_eq!(report.points[0].position, [1.0, 0.0]);
    second.left_offset = f64::NAN;
    let failure = intersect_planar_curves(&first, &second, options()).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidInput);
    assert!(failure.source().is_none());
    assert!(failure.to_string().contains("NaN"));
    assert_eq!(report.points[0].parameters, [1.0, 1.0]);
    assert_eq!(
        intersect_planar_curves(&first, &before[1], options()).unwrap(),
        report
    );
}
#[test]
fn typed_native_failure_keeps_its_source_and_does_not_publish_a_report() {
    let invalid = line([0.0; 2], [0.0; 2]);
    let failure = intersect_planar_curves(&invalid, &circle(true), options()).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::Kernel);
    assert!(failure.source().is_some());
    assert!(
        failure
            .to_string()
            .contains(&failure.source().unwrap().to_string())
    );
}
#[test]
fn conic_senses_trim_bounds_and_left_offset_are_native_parameters() {
    for counterclockwise in [false, true] {
        let mut arc = circle(counterclockwise);
        arc.bounds = Some([0.0, std::f64::consts::FRAC_PI_2]);
        let vertical = line([1.0, 0.0], [0.0, 1.0]);
        let report = intersect_planar_curves(&vertical, &arc, options()).unwrap();
        assert_eq!(report.points.len(), 1);
        let sign = if counterclockwise { 1.0 } else { -1.0 };
        assert!((report.points[0].position[1] - sign * 3.0_f64.sqrt()).abs() < 1e-8);
    }
    let mut horizontal = line([0.0; 2], [1.0, 0.0]);
    horizontal.left_offset = 0.25;
    horizontal.bounds = Some([-2.0, 2.0]);
    let report =
        intersect_planar_curves(&horizontal, &line([1.0, 0.0], [0.0, 1.0]), options()).unwrap();
    assert_eq!(report.points[0].position, [1.0, 0.25]);
}
#[test]
fn rational_spline_and_ellipse_return_analytic_points_with_measured_residuals() {
    let axis = line([0.0, 1.0], [1.0, 0.0]);
    let ellipse = definition(PlanarGeometry::Ellipse {
        frame: conic_frame(true),
        radii: [3.0, 2.0],
    });
    for (curve, count) in [(spline(), 1), (ellipse, 2)] {
        let report = intersect_planar_curves(&axis, &curve, options()).unwrap();
        assert_eq!(report.points.len(), count);
        assert!(report.overlaps.is_empty());
        assert!(
            report
                .points
                .iter()
                .flat_map(|hit| hit.residuals)
                .all(|distance| distance <= options().maximum_point_residual)
        );
    }
    let report = intersect_planar_curves(&axis, &spline(), options()).unwrap();
    assert!((report.points[0].position[0] - 3.0_f64.sqrt()).abs() < 1e-8);
}
#[test]
fn bounded_and_unbounded_overlaps_keep_incidence_and_direction() {
    let mut first = line([0.0; 2], [1.0, 0.0]);
    let mut second = line([0.0; 2], [-1.0, 0.0]);
    let unbounded = intersect_planar_curves(&first, &second, options()).unwrap();
    assert_eq!(
        unbounded.overlaps,
        [PlanarOverlap {
            first: None,
            last: None,
            opposite: true
        }]
    );
    first.bounds = Some([0.0, 2.0]);
    second.bounds = Some([-3.0, -1.0]);
    let bounded = intersect_planar_curves(&first, &second, options()).unwrap();
    assert_eq!(bounded.overlaps.len(), 1);
    let overlap = &bounded.overlaps[0];
    assert!(overlap.opposite);
    assert_eq!(overlap.first.as_ref().unwrap().parameters, [1.0, -1.0]);
    assert_eq!(overlap.last.as_ref().unwrap().parameters, [2.0, -2.0]);
}

fn conic_frame(counterclockwise: bool) -> PlanarConicFrame {
    PlanarConicFrame {
        center: [0.0; 2],
        axis: [1.0, 0.0],
        counterclockwise,
    }
}
#[test]
fn unbounded_offset_keeps_the_native_domain_failure_without_a_retry() {
    let mut offset = line([0.0; 2], [1.0, 0.0]);
    offset.left_offset = 0.25;
    let failure =
        intersect_planar_curves(&offset, &line([1.0, 0.0], [0.0, 1.0]), options()).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::Kernel);
    assert!(
        failure
            .source()
            .unwrap()
            .to_string()
            .contains("Domaine sur courbe incorrect")
    );
}
#[test]
fn all_option_fields_require_finite_positive_values() {
    for value in [0.0, -1.0, f64::NAN, f64::INFINITY] {
        for index in 0..3 {
            assert_invalid_option(index, value);
        }
    }
}
fn assert_invalid_option(index: usize, value: f64) {
    let mut limits = options();
    match index {
        0 => limits.confusion_tolerance = value,
        1 => limits.intersection_tolerance = value,
        _ => limits.maximum_point_residual = value,
    }
    let failure = intersect_planar_curves(&circle(true), &spline(), limits).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidInput);
    assert!(failure.to_string().contains("expected finite positive"));
}
#[test]
fn invalid_coordinate_radius_and_trim_values_are_declined() {
    let mut inputs = vec![
        line([f64::NAN, 0.0], [1.0, 0.0]),
        line([0.0; 2], [f64::INFINITY, 0.0]),
        definition(PlanarGeometry::Circle {
            frame: conic_frame(true),
            radius: -1.0,
        }),
    ];
    for bounds in [[1.0, 0.0], [1.0, 1.0], [0.0, f64::INFINITY]] {
        let mut curve = circle(true);
        curve.bounds = Some(bounds);
        inputs.push(curve);
    }
    for input in inputs {
        let failure = intersect_planar_curves(&input, &spline(), options()).unwrap_err();
        assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidInput);
        assert!(failure.to_string().contains("first curve"));
    }
}
#[test]
fn spline_constructor_failure_retains_cause_and_operand_context() {
    let mut invalid = spline();
    let PlanarGeometry::Spline(definition) = &mut invalid.geometry else {
        unreachable!()
    };
    definition.multiplicities.clear();
    let failure = intersect_planar_curves(&circle(true), &invalid, options()).unwrap_err();
    assert_eq!(failure.kind(), PlanarCurveFailureKind::Kernel);
    assert!(failure.source().is_some());
    assert!(failure.to_string().contains("second curve"));
}
#[test]
fn separated_curves_return_a_completed_empty_report() {
    let report =
        intersect_planar_curves(&circle(true), &line([0.0, 3.0], [1.0, 0.0]), options()).unwrap();
    assert_eq!(
        report,
        PlanarIntersectionReport {
            points: vec![],
            overlaps: vec![]
        }
    );
}
#[test]
fn endpoint_and_tangent_metadata_survive_conversion() {
    let mut arc = circle(true);
    arc.bounds = Some([0.0, std::f64::consts::FRAC_PI_2]);
    let report = intersect_planar_curves(&arc, &line([2.0, 0.0], [0.0, 1.0]), options()).unwrap();
    assert_eq!(report.points.len(), 1);
    assert_eq!(
        report.points[0].transitions[0].position,
        CurvePosition::Start
    );
    assert!(matches!(
        report.points[0].transitions[0].kind,
        TransitionKind::Touch { tangent: true, .. }
    ));
}
#[test]
fn periodic_arc_across_the_seam_keeps_its_unwrapped_parameters() {
    let mut arc = circle(true);
    arc.bounds = Some([1.5 * std::f64::consts::PI, 2.5 * std::f64::consts::PI]);
    let report = intersect_planar_curves(&arc, &line([0.0; 2], [1.0, 0.0]), options()).unwrap();
    assert_eq!(report.points.len(), 1);
    assert!((report.points[0].parameters[0] - std::f64::consts::TAU).abs() < 1e-8);
    assert!((report.points[0].position[0] - 2.0).abs() < 1e-8);
}
#[test]
fn unweighted_spline_uses_the_shared_native_constructor() {
    let mut curve = spline();
    let PlanarGeometry::Spline(spline) = &mut curve.geometry else {
        unreachable!()
    };
    spline.weights = None;
    spline.poles = vec![[0.0, 0.25], [0.5, -0.25], [1.0, 0.25]];
    let report = intersect_planar_curves(&curve, &line([0.0; 2], [1.0, 0.0]), options()).unwrap();
    assert_eq!(report.points.len(), 1);
    assert!((report.points[0].parameters[0] - 0.5).abs() < 1e-8);
    assert!((report.points[0].position[0] - 0.5).abs() < 1e-8);
}
#[test]
fn source_operand_swap_swaps_parameter_and_transition_ownership() {
    let first = line([0.0; 2], [1.0, 0.0]);
    let second = line([2.0, -3.0], [0.0, 1.0]);
    let forward = intersect_planar_curves(&first, &second, options()).unwrap();
    let reverse = intersect_planar_curves(&second, &first, options()).unwrap();
    assert_eq!(forward.points[0].parameters, [2.0, 3.0]);
    assert_eq!(reverse.points[0].parameters, [3.0, 2.0]);
    assert_eq!(forward.points[0].position, reverse.points[0].position);
    assert_eq!(
        forward.points[0].transitions[0],
        reverse.points[0].transitions[1]
    );
    assert_eq!(
        forward.points[0].transitions[1],
        reverse.points[0].transitions[0]
    );
}
#[test]
fn trims_do_not_extend_bounded_splines_or_repeat_a_period() {
    let mut spline = spline();
    spline.bounds = Some([-1e-10, 0.5]);
    let mut repeated = circle(true);
    repeated.bounds = Some([0.0, 2.0 * std::f64::consts::TAU]);
    for invalid in [spline, repeated] {
        let failure =
            intersect_planar_curves(&invalid, &line([0.0; 2], [1.0, 0.0]), options()).unwrap_err();
        assert_eq!(failure.kind(), PlanarCurveFailureKind::InvalidInput);
        assert!(failure.to_string().contains("bounds"));
    }
}
