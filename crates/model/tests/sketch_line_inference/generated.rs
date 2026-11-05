// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn rotated_line(angle: f64, start: [f64; 2]) -> [[f64; 2]; 2] {
    [
        start,
        [start[0] + 4.0 * angle.cos(), start[1] + 4.0 * angle.sin()],
    ]
}
fn assert_relative_rotation(angle: f64, offset: f64, kind: ConstraintInferenceKind) {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, rotated_line(angle, [10.0, 10.0]));
    let before = sketch.clone();
    for miss in [-1.0_f64, 0.0, 1.0] {
        let endpoints = rotated_line(angle + offset + miss.to_radians(), [0.0; 2]);
        let result = direction(&sketch, endpoints, PRIORITY_PARALLEL_PERPENDICULAR);
        assert_eq!(
            (result.kind, result.target),
            (kind, Some(SketchEntityId(5)))
        );
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn generated_rotations_reproduce_analytic_parallel_and_perpendicular_families() {
    for degrees in (0..360).step_by(15) {
        let angle = f64::from(degrees).to_radians();
        assert_relative_rotation(angle, 0.0, INFER_PARALLEL);
        assert_relative_rotation(angle, std::f64::consts::FRAC_PI_2, INFER_PERPENDICULAR);
    }
}
fn scaled_placement(scale: f64) -> SketchLinePlacementInference {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    add_line(
        &mut sketch,
        5,
        [[10.0 / scale, 20.0 / scale], [14.0 / scale, 24.0 / scale]],
    );
    let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
    let endpoints = [[0.0005 / scale, 0.0005 / scale], [4.0 / scale, 4.0 / scale]];
    sketch
        .infer_line_placement(
            endpoints,
            &default_sketch_settings(),
            SketchInferencePolicy::for_units(&units),
        )
        .unwrap()
}
#[test]
fn physical_snap_and_direction_targets_survive_working_scale_changes() {
    for scale in [0.0001, 0.1, 1.0, 2.0, 10000.0] {
        let result = scaled_placement(scale);
        assert_eq!(
            result.points,
            [SketchEndpointInference {
                endpoint: 0,
                target: SketchEntityId(1)
            }]
        );
        assert_eq!(result.candidate_endpoints[0], [0.0; 2]);
        assert_eq!(result.direction.unwrap().kind, INFER_PARALLEL);
        assert_eq!(result.direction.unwrap().target, Some(SketchEntityId(5)));
    }
}
#[test]
fn both_endpoints_can_propose_the_same_existing_target_without_fresh_point_candidates() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(7), [0.0; 2]).unwrap();
    let report = sketch
        .infer_line_placement(
            [[0.0001, 0.0], [0.0, 0.0001]],
            &default_sketch_settings(),
            policy(),
        )
        .unwrap();
    assert_eq!(report.points.len(), 2);
    for (index, suggestion) in report.points.iter().enumerate() {
        assert_eq!(
            (suggestion.endpoint, suggestion.target),
            (index, SketchEntityId(7))
        );
    }
    assert_eq!(report.candidate_endpoints, [[0.0; 2]; 2]);
    assert!(report.direction.is_none());
}
