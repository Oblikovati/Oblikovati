// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::error::Error;
fn attempt_caught(
    history: &mut DocumentEditHistory,
    request: SketchLineCreation,
    policy: SketchInferencePolicy,
) -> String {
    let token = history.begin("Catch line failure").unwrap();
    let cause = history
        .edit(&token, |snapshot, ids| -> Result<String, String> {
            let before = snapshot.sketches.clone();
            let sketch = DocumentSketchId(1);
            let options = Default::default();
            let cause = snapshot
                .create_inferred_sketch_line(sketch, &request, policy, options, ids)
                .unwrap_err();
            assert_eq!(snapshot.sketches, before);
            assert!(cause.source().is_some());
            Ok(cause.to_string())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    cause
}
fn policy() -> SketchInferencePolicy {
    SketchInferencePolicy::for_units(&UnitsOfMeasure::default())
}
#[test]
fn nonconverged_source_refuses_inference_before_any_reservations() {
    let sketch = contradictory_source();
    let mut history = owner(sketch);
    let before = history.sketch_reservations();
    let cause = attempt_caught(&mut history, input([[10.0, 10.0], [12.0, 10.0]]), policy());
    assert!(
        cause.contains("expected converged current coordinates"),
        "{cause}"
    );
    assert_eq!(history.sketch_reservations(), before);
}
#[test]
fn invalid_coordinates_and_policy_fail_before_identity_reservation() {
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let mut history = owner(reference());
        let before = history.sketch_reservations();
        let cause = attempt_caught(&mut history, input([[value, 0.0], [1.0, 0.0]]), policy());
        assert!(cause.contains("finite"), "{cause}");
        assert_eq!(history.sketch_reservations(), before);
    }
    let mut history = owner(reference());
    let before = history.sketch_reservations();
    let invalid = SketchInferencePolicy {
        snap_distance_working: 0.0,
        ..policy()
    };
    assert!(!attempt_caught(&mut history, input([[0.0; 2], [1.0, 0.0]]), invalid).is_empty());
    assert_eq!(history.sketch_reservations(), before);
}
#[test]
fn exhausted_curve_identity_keeps_consumed_points_without_publishing_them() {
    let marks = SketchReservationMarks {
        sketch: 1,
        entity: u64::MAX - 2,
        constraints: Default::default(),
        dimension_names: Default::default(),
    };
    let mut history = owner(reference()).with_sketch_reservations(marks).unwrap();
    let before = history.snapshot().sketches.clone();
    let cause = attempt_caught(&mut history, input([[10.0, 10.0], [14.0, 11.0]]), policy());
    assert!(cause.contains("identity"), "{cause}");
    assert_eq!(history.sketch_reservations().entity, u64::MAX);
    assert_eq!(history.snapshot().sketches, before);
}
#[test]
fn second_snap_exhaustion_retains_first_constraint_reservation_without_partial_geometry() {
    let marks = SketchReservationMarks {
        sketch: 1,
        entity: 3,
        constraints: [(1, u64::MAX - 1)].into(),
        dimension_names: Default::default(),
    };
    let mut history = owner(reference()).with_sketch_reservations(marks).unwrap();
    let before = history.snapshot().sketches.clone();
    let cause = attempt_caught(&mut history, input([[0.0, 0.0], [3.0, 2.0]]), policy());
    assert!(cause.contains("1"), "{cause}");
    assert_eq!(history.sketch_reservations().constraints[&1], u64::MAX);
    assert_eq!(history.sketch_reservations().entity, 6);
    assert_eq!(history.snapshot().sketches, before);
}
#[test]
fn undo_and_abort_never_reuse_entity_or_constraint_reservations() {
    let mut history = owner(reference());
    let request = input([[0.0, 0.0], [3.0, 2.0]]);
    let first = create(&mut history, request);
    let marks = history.sketch_reservations();
    history.undo().unwrap();
    assert_eq!(history.sketch_reservations(), marks);
    let second = create_aborted(&mut history, request);
    assert!(second.primitive.entity > first.primitive.entity);
    assert!(second.direction.unwrap().constraint.0 > first.direction.unwrap().constraint.0);
    assert_eq!(geometry(&history).lines().len(), 1);
    assert_eq!(
        history.sketch_reservations().entity,
        second.primitive.entity.0
    );
}

fn create_aborted(
    history: &mut DocumentEditHistory,
    request: SketchLineCreation,
) -> SketchLineCreationReport {
    let token = history.begin("Discard replacement").unwrap();
    let second = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_inferred_sketch_line(
                DocumentSketchId(1),
                &request,
                policy(),
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.abort(&token).unwrap();
    second
}

fn contradictory_source() -> Sketch2D {
    let mut sketch = reference();
    for point in sketch.points().to_vec() {
        sketch
            .add_constraint(
                SketchConstraintId(point.id.0),
                PlanarConstraint::Fix {
                    point: point.id,
                    position: point.position,
                },
            )
            .unwrap();
    }
    let relation = PlanarConstraint::LineHorizontal {
        line: SketchEntityId(3),
    };
    let id = SketchConstraintId(3);
    sketch.add_constraint(id, relation).unwrap();
    sketch
}
