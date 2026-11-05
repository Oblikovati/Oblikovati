// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    ConstraintInferencePriority, PRIORITY_HORIZONTAL_VERTICAL, PRIORITY_PARALLEL_PERPENDICULAR,
};
#[test]
fn go_horizontal_application_fixture_drives_the_created_line() {
    let mut history = owner(Sketch2D::default());
    let report = create(&mut history, input([[0.0, 0.0], [4.0, 0.05]]));
    assert_eq!(report.direction.unwrap().inference.kind, INFER_HORIZONTAL);
    assert_eq!(geometry(&history).geometric_constraints().count(), 1);
    let outcome = geometry(&history)
        .solve(&history.snapshot().parameters, Default::default(), || false)
        .unwrap();
    let points = outcome.candidate().points();
    assert!((points[1].position[1] - points[0].position[1]).abs() < 1e-9);
}
#[test]
fn go_priority_fixture_selects_horizontal_or_parallel_on_the_same_coordinates() {
    for (priority, expected) in [
        (PRIORITY_HORIZONTAL_VERTICAL, INFER_HORIZONTAL),
        (PRIORITY_PARALLEL_PERPENDICULAR, INFER_PARALLEL),
    ] {
        let mut history = priority_owner(priority);
        let report = create(&mut history, input([[0.0, 0.0], [4.0, 0.04]]));
        assert_eq!(report.direction.unwrap().inference.kind, expected);
        assert_direction_relation(geometry(&history), report.primitive.entity, expected);
    }
}
fn priority_owner(priority: ConstraintInferencePriority) -> DocumentEditHistory {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0, 5.0]).unwrap();
    sketch.add_point(SketchEntityId(2), [4.0, 5.02]).unwrap();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let mut history = owner(sketch);
    history
        .refresh(|snapshot| -> Result<(), String> {
            let mut settings = default_sketch_settings();
            settings.constraint_priority = priority;
            snapshot.sketch_settings = Some(settings);
            Ok(())
        })
        .unwrap();
    history
}
#[test]
fn go_endpoint_snap_fixture_keeps_exact_anchor_coordinates() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [2.0, 1.0]).unwrap();
    let mut history = owner(sketch);
    let report = create(&mut history, input([[2.0005, 1.0005], [5.0, 4.0]]));
    assert_eq!(report.point_constraints.len(), 1);
    assert_eq!(report.inference.points[0].target, SketchEntityId(1));
    let start = geometry(&history)
        .point(report.primitive.points[0])
        .unwrap()
        .position;
    assert_eq!(start, [2.0, 1.0]);
}
#[test]
fn inference_disabled_creates_raw_geometry_without_proposals_or_relations() {
    let mut history = owner(reference());
    history
        .refresh(|snapshot| -> Result<(), String> {
            let mut settings = default_sketch_settings();
            settings.infer_constraints = false;
            snapshot.sketch_settings = Some(settings);
            Ok(())
        })
        .unwrap();
    let report = create(&mut history, input([[0.0001, 0.0001], [3.0001, 2.0001]]));
    assert!(report.inference.points.is_empty());
    assert!(report.inference.direction.is_none());
    assert!(geometry(&history).constraints().is_empty());
}
