// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::{PlacedSketchHealth, SketchEvaluation};
#[test]
fn snapshot_primitive_failure_is_atomic_before_the_outer_command_discards_it() {
    let mut session = exhausted_session(u64::MAX - 1);
    let sketch = DocumentSketchId(1);
    let error = session
        .run("Exhaust primitive", |snapshot, reservations| {
            let before = snapshot.sketches.clone();
            let result = snapshot.create_sketch_primitive(sketch, &circle(2.0, true), reservations);
            assert!(matches!(
                result,
                Err(SketchPrimitiveCreationError::Identity(_))
            ));
            assert!(snapshot.sketches.same_evaluated_state(&before));
            result
        })
        .unwrap_err();
    assert!(error.to_string().contains("identity"));
}
#[test]
fn degenerate_arc_records_a_named_solve_failure_instead_of_claiming_valid_geometry() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let request = SketchPrimitiveCreation::Arc {
        center: SketchPointInput::New([0.0; 2]),
        endpoints: [SketchPointInput::New([0.0; 2]); 2],
        counter_clockwise: true,
        construction: false,
    };
    session.create_sketch_primitive(sketch, &request).unwrap();
    let snapshot = session.snapshot();
    assert_failed_arc(snapshot, sketch);
    assert!(snapshot.require_resolved_sketch(sketch).is_err());
}
#[test]
fn primitive_creation_observer_receives_one_commit_and_current_snapshot() {
    let mut session = prepared_session(None, 0);
    let sketch = DocumentSketchId(1);
    session
        .create_sketch_primitive(sketch, &circle(2.0, true))
        .unwrap();
    let observations = &session.observer().observations;
    assert_eq!(observations.len(), 3);
    assert!(
        matches!(&observations[0], Observation::Before(payload, _) if payload.type_ == EVENT_TRANSACTION_COMMITTED)
    );
    assert!(
        matches!(&observations[2], Observation::After(payload, _) if payload.type_ == EVENT_TRANSACTION_COMMITTED)
    );
    assert!(matches!(&observations[1], Observation::Changed(17, _)));
    assert_numeric_verification(&session, sketch, 3);
}
#[test]
fn existing_circle_center_can_be_shared_and_point_creation_has_no_curve_role() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let report = session
        .create_sketch_primitive(sketch, &point([4.0, 5.0]))
        .unwrap();
    let request = SketchPrimitiveCreation::Circle {
        center: SketchPointInput::Owned(report.entity),
        radius: 3.0,
        construction: false,
    };
    let circle = session.create_sketch_primitive(sketch, &request).unwrap();
    assert_eq!(circle.created, [SketchEntityId(2)]);
    assert_eq!(circle.points, [report.entity]);
    assert_point_and_circle_identity(&session, sketch, report.entity, circle.entity);
}

fn assert_failed_arc(snapshot: &DocumentSnapshot, sketch: DocumentSketchId) {
    match snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .health(&snapshot.parameters)
    {
        PlacedSketchHealth::Current(SketchEvaluation::Failed(cause)) => {
            assert!(cause.to_string().contains("radius"))
        }
        unexpected => panic!("degenerate arc {unexpected:?}: expected current named failure"),
    }
}
fn assert_point_and_circle_identity(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    point: SketchEntityId,
    circle: SketchEntityId,
) {
    assert!(!geometry(session, sketch).is_construction(point).unwrap());
    assert_eq!(
        geometry(session, sketch).circle(circle).unwrap().center,
        point
    );
}
#[test]
fn fully_shared_arc_keeps_all_defining_point_ids_and_an_independent_direction() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let first = session
        .create_sketch_primitive(sketch, &arc(false, false))
        .unwrap();
    let request = shared_arc(&first.points);
    let second = session.create_sketch_primitive(sketch, &request).unwrap();
    assert_eq!(second.created, [SketchEntityId(5)]);
    assert_eq!(second.points, first.points);
    assert_arc_direction(&session, sketch, second.entity, true);
    assert_arc_direction(&session, sketch, first.entity, false);
}
fn shared_arc(points: &[SketchEntityId]) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Arc {
        center: SketchPointInput::Owned(points[0]),
        endpoints: [
            SketchPointInput::Owned(points[1]),
            SketchPointInput::Owned(points[2]),
        ],
        counter_clockwise: true,
        construction: true,
    }
}
fn assert_arc_direction(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    entity: SketchEntityId,
    expected: bool,
) {
    assert_eq!(
        geometry(session, sketch)
            .arc(entity)
            .unwrap()
            .counter_clockwise,
        expected
    );
}
