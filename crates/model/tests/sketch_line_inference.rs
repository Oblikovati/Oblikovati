// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use oblikovati_model::{
    sketch::*,
    sketch_settings::{SketchSettings, default_sketch_settings},
    units::UnitsOfMeasure,
};
fn policy() -> SketchInferencePolicy {
    SketchInferencePolicy::for_units(&UnitsOfMeasure::default())
}
fn options(priority: ConstraintInferencePriority, constrain: bool) -> SketchSettings {
    let mut settings = default_sketch_settings();
    settings.constraint_priority = priority;
    settings.auto_apply_constraints = constrain;
    settings
}
fn add_line(sketch: &mut Sketch2D, id: u64, coordinates: [[f64; 2]; 2]) {
    sketch
        .add_point(SketchEntityId(id + 1), coordinates[0])
        .unwrap();
    sketch
        .add_point(SketchEntityId(id + 2), coordinates[1])
        .unwrap();
    sketch
        .add_line(
            SketchEntityId(id),
            [SketchEntityId(id + 1), SketchEntityId(id + 2)],
        )
        .unwrap();
}
fn infer(
    sketch: &Sketch2D,
    endpoints: [[f64; 2]; 2],
    priority: ConstraintInferencePriority,
) -> SketchLinePlacementInference {
    sketch
        .infer_line_placement(endpoints, &options(priority, false), policy())
        .unwrap()
}
fn direction(
    sketch: &Sketch2D,
    endpoints: [[f64; 2]; 2],
    priority: ConstraintInferencePriority,
) -> SketchDirectionInference {
    infer(sketch, endpoints, priority).direction.unwrap()
}
#[test]
fn reference_policy_has_go_angle_and_cm_scale_distance() {
    assert_eq!(
        policy().angle_tolerance_radians,
        3.0 * std::f64::consts::PI / 180.0
    );
    assert_eq!(policy().snap_distance_working, 0.001);
    for scale in [0.1, 2.0, 100.0] {
        let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
        assert_eq!(
            SketchInferencePolicy::for_units(&units).snap_distance_working,
            0.001 / scale
        );
    }
}
#[test]
fn axis_quadrants_and_endpoint_reversal_preserve_direction_kind() {
    for sign in [-1.0, 1.0] {
        let sketch = Sketch2D::default();
        let horizontal = [[0.0, 0.0], [5.0 * sign, 0.05 * sign]];
        assert_eq!(
            direction(&sketch, horizontal, PRIORITY_NONE).kind,
            INFER_HORIZONTAL
        );
        assert_eq!(
            direction(
                &sketch,
                [[0.0, 0.0], [0.05 * sign, 5.0 * sign]],
                PRIORITY_NONE
            )
            .kind,
            INFER_VERTICAL
        );
    }
}
#[test]
fn axis_quality_reproduces_go_integer_truncation() {
    let sketch = Sketch2D::default();
    assert_eq!(
        direction(&sketch, [[0.0; 2], [5.0, 0.05]], PRIORITY_NONE).score,
        80
    );
    assert_eq!(
        direction(&sketch, [[0.0; 2], [5.0, 0.0]], PRIORITY_NONE).score,
        100
    );
    assert_eq!(
        direction(&sketch, [[0.0; 2], [0.0, -5.0]], PRIORITY_NONE).score,
        100
    );
}
#[test]
fn no_reference_diagonal_and_zero_length_have_no_direction() {
    let sketch = Sketch2D::default();
    for endpoints in [[[0.0; 2], [5.0, 5.0]], [[1.0, 1.0]; 2]] {
        assert_eq!(infer(&sketch, endpoints, PRIORITY_NONE).direction, None);
    }
}
#[test]
fn horizontal_vertical_priority_wins_over_a_better_relative_match() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0, 5.0], [4.0, 5.02]]);
    let endpoints = [[0.0; 2], [4.0, 0.04]];
    let suggestion = direction(&sketch, endpoints, PRIORITY_HORIZONTAL_VERTICAL);
    assert_eq!(
        (suggestion.kind, suggestion.target),
        (INFER_HORIZONTAL, None)
    );
    let suggestion = direction(&sketch, endpoints, PRIORITY_PARALLEL_PERPENDICULAR);
    assert_eq!(
        (suggestion.kind, suggestion.target),
        (INFER_PARALLEL, Some(SketchEntityId(5)))
    );
    assert_eq!(suggestion.score, 90);
}
#[test]
fn no_family_priority_uses_quality_and_prefers_axial_on_integer_ties() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0, 5.0], [4.0, 5.02]]);
    assert_eq!(
        direction(&sketch, [[0.0; 2], [4.0, 0.04]], PRIORITY_NONE).kind,
        INFER_PARALLEL
    );
    let mut exact = Sketch2D::default();
    add_line(&mut exact, 5, [[0.0, 5.0], [4.0, 5.0]]);
    assert_eq!(
        direction(&exact, [[0.0; 2], [4.0, 0.0]], PRIORITY_NONE).kind,
        INFER_HORIZONTAL
    );
}
#[test]
fn relative_direction_is_orientation_independent_and_reports_the_owned_line() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 9, [[4.0, 9.0], [0.0, 5.0]]);
    for endpoint in [[4.0, 4.0], [-4.0, -4.0]] {
        let proposal = direction(&sketch, [[0.0; 2], endpoint], PRIORITY_NONE);
        assert_eq!(
            (proposal.kind, proposal.target),
            (INFER_PARALLEL, Some(SketchEntityId(9)))
        );
    }
    assert_eq!(
        direction(&sketch, [[0.0; 2], [-4.0, 4.0]], PRIORITY_NONE).kind,
        INFER_PERPENDICULAR
    );
}
#[test]
fn equal_relative_scores_select_the_first_authored_line_not_the_smallest_id() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 17, [[0.0, 5.0], [4.0, 9.0]]);
    add_line(&mut sketch, 5, [[0.0, 10.0], [4.0, 14.0]]);
    assert_eq!(
        direction(&sketch, [[0.0; 2], [4.0, 4.0]], PRIORITY_NONE).target,
        Some(SketchEntityId(17))
    );
}
#[test]
fn a_later_better_relative_candidate_replaces_the_first() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 17, [[0.0, 5.0], [4.0, 9.02]]);
    add_line(&mut sketch, 5, [[0.0, 10.0], [4.0, 14.0]]);
    assert_eq!(
        direction(&sketch, [[0.0; 2], [4.0, 4.0]], PRIORITY_NONE).target,
        Some(SketchEntityId(5))
    );
}
#[test]
fn point_ties_select_the_last_authored_point_and_nearest_wins_over_order() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(17), [0.0; 2]).unwrap();
    sketch.add_point(SketchEntityId(5), [0.0; 2]).unwrap();
    let report = infer(&sketch, [[0.0005, 0.0], [4.0, 4.0]], PRIORITY_NONE);
    assert_eq!(
        report.points,
        [SketchEndpointInference {
            endpoint: 0,
            target: SketchEntityId(5)
        }]
    );
    sketch.add_point(SketchEntityId(3), [0.0014, 0.0]).unwrap();
    assert_eq!(
        infer(&sketch, [[0.0005, 0.0], [4.0, 4.0]], PRIORITY_NONE).points,
        report.points
    );
}
#[test]
fn snap_boundary_is_closed_and_fresh_endpoints_are_never_targets() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(7), [0.0; 2]).unwrap();
    assert_eq!(
        infer(&sketch, [[0.001, 0.0], [4.0, 4.0]], PRIORITY_NONE)
            .points
            .len(),
        1
    );
    assert!(
        infer(&sketch, [[0.001000000001, 0.0], [4.0, 4.0]], PRIORITY_NONE)
            .points
            .is_empty()
    );
}
#[test]
fn constraining_uses_virtual_snaps_before_direction_and_query_never_mutates() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0, -0.0005]).unwrap();
    let before = sketch.clone();
    let endpoints = [[0.0, 0.0], [0.01, 0.00002]];
    let audit = infer(&sketch, endpoints, PRIORITY_NONE);
    let applied = sketch
        .infer_line_placement(endpoints, &options(PRIORITY_NONE, true), policy())
        .unwrap();
    assert_eq!(audit.candidate_endpoints, endpoints);
    assert_eq!(applied.candidate_endpoints[0], [0.0, -0.0005]);
    assert!(audit.direction.unwrap().score > applied.direction.unwrap().score);
    assert_eq!(audit.points, applied.points);
    assert!(sketch.same_recipe(&before));
}
#[test]
fn inference_disabled_returns_no_proposals_or_virtual_changes() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    let mut settings = default_sketch_settings();
    settings.infer_constraints = false;
    let endpoints = [[0.0001, 0.0], [4.0, 0.0]];
    let report = sketch
        .infer_line_placement(endpoints, &settings, policy())
        .unwrap();
    assert_eq!(report.candidate_endpoints, endpoints);
    assert!(report.points.is_empty() && report.direction.is_none());
}
#[path = "sketch_line_inference/ranges.rs"]
mod ranges;
#[test]
fn endpoint_reversal_retains_axis_kind_and_quality() {
    let sketch = Sketch2D::default();
    let endpoints = [[0.0; 2], [-5.0, -0.05]];
    assert_eq!(
        direction(&sketch, endpoints, PRIORITY_NONE),
        direction(&sketch, [endpoints[1], endpoints[0]], PRIORITY_NONE)
    );
}
#[test]
fn two_fresh_endpoints_are_not_each_others_snap_targets() {
    let report = infer(
        &Sketch2D::default(),
        [[0.0; 2], [0.0001, 0.0]],
        PRIORITY_NONE,
    );
    assert!(report.points.is_empty());
    assert_eq!(report.direction.unwrap().kind, INFER_HORIZONTAL);
}
#[test]
fn preferred_relative_family_wins_even_with_lower_quality() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0, 5.0], [4.0, 5.08]]);
    let endpoints = [[0.0; 2], [4.0, 0.02]];
    let axial = direction(&sketch, endpoints, PRIORITY_HORIZONTAL_VERTICAL);
    let relative = direction(&sketch, endpoints, PRIORITY_PARALLEL_PERPENDICULAR);
    assert_eq!(
        (axial.kind, relative.kind),
        (INFER_HORIZONTAL, INFER_PARALLEL)
    );
    assert!(axial.score > relative.score);
}
#[test]
fn angular_boundary_is_closed_with_zero_quality_then_declines_beyond_it() {
    let sketch = Sketch2D::default();
    let policy = SketchInferencePolicy {
        angle_tolerance_radians: 0.05_f64.atan2(1.0),
        ..policy()
    };
    let settings = default_sketch_settings();
    let at = sketch
        .infer_line_placement([[0.0; 2], [1.0, 0.05]], &settings, policy)
        .unwrap();
    assert_eq!(
        (at.direction.unwrap().kind, at.direction.unwrap().score),
        (INFER_HORIZONTAL, 0)
    );
    let beyond = sketch
        .infer_line_placement([[0.0; 2], [1.0, 0.05001]], &settings, policy)
        .unwrap();
    assert!(beyond.direction.is_none());
}
#[path = "sketch_line_inference/generated.rs"]
mod generated;
#[test]
fn construction_lines_are_still_valid_direction_reference_candidates() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0, 5.0], [4.0, 9.0]]);
    sketch
        .set_curve_construction(&[SketchEntityId(5)], true)
        .unwrap();
    let before = sketch.clone();
    assert_eq!(
        direction(&sketch, [[0.0; 2], [4.0, 4.0]], PRIORITY_NONE).target,
        Some(SketchEntityId(5))
    );
    assert!(sketch.same_recipe(&before));
}
