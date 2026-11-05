// SPDX-License-Identifier: GPL-2.0-only
//! Released-kernel prerequisites for analytic sketch trim and extend.
use rustycad_tkernel::{ncollection::Array1, standard::Handle};
use rustycad_tkg2d::geom2d::{
    Geom2dBSplineCurve, Geom2dCircle, Geom2dCurve, Geom2dEllipse, Geom2dLine, Geom2dTrimmedCurve,
};
use rustycad_tkgeomalgo::{geom2d_api::InterCurveCurve, int_res2d::IntersectionPoint};
use rustycad_tkmath::gp::{Axis2d, Direction2, Ellipse2, Point2};
#[path = "planar_intersections/mod.rs"]
mod support;
use support::*;
#[test]
fn crossing_lines_report_completion_and_parameters_on_both_supports() {
    assert_points(
        &line([0.0, 0.0], [1.0, 0.0]),
        &line([1.0, -1.0], [0.0, 1.0]),
        &[[1.0, 0.0]],
    );
}
#[test]
fn ellipse_crossings_tangency_and_empty_results_keep_distinct_completed_outcomes() {
    let curve = ellipse([3.0, 2.0]);
    assert_points(
        &line([0.0, 0.0], [1.0, 0.0]),
        &curve,
        &[[-3.0, 0.0], [3.0, 0.0]],
    );
    assert_points(&line([0.0, 2.0], [1.0, 0.0]), &curve, &[[0.0, 2.0]]);
    assert_points(&line([0.0, 3.0], [1.0, 0.0]), &curve, &[]);
}
#[test]
fn trimmed_quarter_circle_keeps_its_sweep_and_endpoint_crossings() {
    let full = circle(2.0);
    let quarter = trim(&full, [0.0, std::f64::consts::FRAC_PI_2]);
    assert_points(
        &line([1.0, 0.0], [0.0, 1.0]),
        &quarter,
        &[[1.0, 3.0_f64.sqrt()]],
    );
    assert_points(&line([2.0, 0.0], [0.0, 1.0]), &quarter, &[[2.0, 0.0]]);
    assert_points(&line([-1.0, 0.0], [0.0, 1.0]), &quarter, &[]);
}
#[test]
fn rational_quadratic_circle_crossing_uses_the_analytic_curve() {
    let curve = spline(
        &[[2.0, 0.0], [2.0, 2.0], [0.0, 2.0]],
        Some(&[1.0, std::f64::consts::FRAC_1_SQRT_2, 1.0]),
    );
    assert_points(
        &line([0.0, 1.0], [1.0, 0.0]),
        &curve,
        &[[3.0_f64.sqrt(), 1.0]],
    );
}
#[test]
fn cubic_three_crossing_roots_are_not_lost() {
    let cubic = spline(
        &[
            [0.0, -0.08],
            [1.0 / 3.0, 0.14],
            [2.0 / 3.0, -0.14],
            [1.0, 0.08],
        ],
        None,
    );
    let axis = line([0.0; 2], [1.0, 0.0]);
    assert_points(&axis, &cubic, &[[0.2, 0.0], [0.5, 0.0], [0.8, 0.0]]);
}
#[test]
fn quadratic_tangent_root_is_not_lost() {
    let axis = line([0.0; 2], [1.0, 0.0]);
    let tangent = spline(&[[0.0, 0.25], [0.5, -0.25], [1.0, 0.25]], None);
    assert_points(&axis, &tangent, &[[0.5, 0.0]]);
}
#[test]
fn overlapping_segments_return_bounded_incidence_instead_of_an_empty_point_set() {
    let support = line([0.0; 2], [1.0, 0.0]);
    let first = trim(&support, [0.0, 2.0]);
    let second = trim(&support, [1.0, 3.0]);
    let result = InterCurveCurve::between(&first, &second, 1e-9).unwrap();
    assert!(result.intersector().is_done());
    assert_eq!(result.nb_segments(), 1);
    let overlap = result.intersector().segment(1).unwrap();
    assert!(overlap.has_first_point() && overlap.has_last_point());
    assert!(!overlap.is_opposite());
    assert_incidence(&first, &second, overlap.first_point().unwrap(), [1.0, 0.0]);
    assert_incidence(&first, &second, overlap.last_point().unwrap(), [2.0, 0.0]);
}
#[test]
fn coincident_infinite_lines_retain_unbounded_overlap() {
    let first = line([0.0; 2], [1.0, 0.0]);
    let second = line([0.0; 2], [-1.0, 0.0]);
    let result = InterCurveCurve::between(&first, &second, 1e-9).unwrap();
    assert!(result.intersector().is_done());
    assert_eq!(result.intersector().nb_points().unwrap(), 0);
    assert_eq!(result.intersector().nb_segments().unwrap(), 1);
    let overlap = result.intersector().segment(1).unwrap();
    assert!(!overlap.has_first_point() && !overlap.has_last_point());
    assert!(overlap.is_opposite());
}
#[test]
fn incomplete_counts_must_not_be_read_as_a_completed_empty_intersection() {
    let pending = InterCurveCurve::new();
    assert!(!pending.intersector().is_done());
    assert_eq!(pending.nb_points(), 0);
    assert_eq!(pending.nb_segments(), 0);
    assert!(pending.intersector().nb_points().is_err());
    assert!(pending.intersector().nb_segments().is_err());
}
#[test]
fn failed_reinitialization_must_not_publish_the_previous_successful_result() {
    let first = line([0.0; 2], [1.0, 0.0]);
    let second = line([1.0, -1.0], [0.0, 1.0]);
    let mut result = InterCurveCurve::between(&first, &second, 1e-9).unwrap();
    let failure = result.init(&Handle::default(), &second, 1e-9).unwrap_err();
    assert!(failure.message().contains("C1 is null"));
    // The released mutable wrapper keeps its old result after this error. A caller
    // must propagate the failure, even when is_done still describes the old call.
    assert!(result.intersector().is_done());
    assert_eq!(result.nb_points(), 1);
    assert_position(result.point(1).unwrap(), [1.0, 0.0]);
}
