// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{
    INFER_PARALLEL, INFER_PERPENDICULAR, INFERENCE_SCOPE_ALL_GEOMETRY,
    INFERENCE_SCOPE_CURRENT_COMMAND, INFERENCE_SCOPE_SELECTED_GEOMETRY,
    PRIORITY_PARALLEL_PERPENDICULAR,
};
use oblikovati_model::{
    sketch::{Sketch2D, SketchEntityId, SketchInferenceCandidates, SketchInferencePolicy},
    sketch_settings::default_sketch_settings,
    units::UnitsOfMeasure,
};

fn two_lines() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, at) in [
        (2, [0., 0.]),
        (3, [4., 4.]),
        (5, [0.0005, 0.]),
        (6, [4.0005, -4.]),
    ] {
        sketch.add_point(SketchEntityId(id), at).unwrap();
    }
    sketch
        .add_line(SketchEntityId(1), [SketchEntityId(2), SketchEntityId(3)])
        .unwrap();
    sketch
        .add_line(SketchEntityId(4), [SketchEntityId(5), SketchEntityId(6)])
        .unwrap();
    sketch
}

#[test]
fn selected_curve_limits_direction_and_its_defining_point_candidates() {
    let sketch = two_lines();
    let mut settings = default_sketch_settings();
    settings.constraint_priority = PRIORITY_PARALLEL_PERPENDICULAR;
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    let endpoints = [[0., 0.], [2., 2.]];
    for (selected, point, kind) in [(1, 2, INFER_PARALLEL), (4, 5, INFER_PERPENDICULAR)] {
        let scope = SketchInferenceCandidates::new(
            INFERENCE_SCOPE_SELECTED_GEOMETRY,
            &[SketchEntityId(selected)],
        )
        .unwrap();
        let result = sketch
            .infer_line_placement_scoped(endpoints, &settings, policy, &scope, &[])
            .unwrap();
        assert_eq!(result.points[0].target, SketchEntityId(point));
        assert_eq!(
            result.direction.unwrap().target,
            Some(SketchEntityId(selected))
        );
        assert_eq!(result.direction.unwrap().kind, kind);
        assert_eq!(
            sketch
                .infer_point_coincidence_scoped([0., 0.], &[], &settings, policy, &scope, &[])
                .unwrap(),
            Some(SketchEntityId(point))
        );
    }
}

#[test]
fn current_command_scope_uses_only_its_supplied_geometry() {
    let sketch = two_lines();
    let mut settings = default_sketch_settings();
    settings.constraint_priority = PRIORITY_PARALLEL_PERPENDICULAR;
    let scope = SketchInferenceCandidates::new(INFERENCE_SCOPE_CURRENT_COMMAND, &[]).unwrap();
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    let endpoints = [[0., 0.], [2., 2.]];
    let empty = sketch
        .infer_line_placement_scoped(endpoints, &settings, policy, &scope, &[])
        .unwrap();
    assert!(empty.points.is_empty());
    assert!(empty.direction.is_none());
    let current = sketch
        .infer_line_placement_scoped(endpoints, &settings, policy, &scope, &[SketchEntityId(1)])
        .unwrap();
    assert_eq!(current.direction.unwrap().target, Some(SketchEntityId(1)));
    assert_eq!(current.points[0].target, SketchEntityId(2));
}

#[test]
fn scopes_refuse_invalid_shapes_and_missing_entities() {
    assert!(SketchInferenceCandidates::new(INFERENCE_SCOPE_SELECTED_GEOMETRY, &[]).is_err());
    assert!(
        SketchInferenceCandidates::new(
            INFERENCE_SCOPE_SELECTED_GEOMETRY,
            &[SketchEntityId(1), SketchEntityId(1)]
        )
        .is_err()
    );
    assert!(
        SketchInferenceCandidates::new(INFERENCE_SCOPE_ALL_GEOMETRY, &[SketchEntityId(1)]).is_err()
    );
    let stale =
        SketchInferenceCandidates::new(INFERENCE_SCOPE_SELECTED_GEOMETRY, &[SketchEntityId(99)])
            .unwrap();
    assert!(stale.validate_geometry(&two_lines()).is_err());
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    assert!(
        two_lines()
            .infer_line_placement_scoped(
                [[0., 0.], [2., 2.]],
                &default_sketch_settings(),
                policy,
                &stale,
                &[]
            )
            .is_err()
    );
}
