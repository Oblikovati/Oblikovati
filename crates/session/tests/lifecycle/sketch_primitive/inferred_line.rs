// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchLineCreation;
fn request() -> SketchLineCreation {
    SketchLineCreation {
        endpoints: [[0.0001, 0.0001], [4.0, 0.01]],
        construction: false,
        centerline: true,
    }
}
#[test]
fn inferred_line_joins_an_open_group_and_recomputes_before_publication() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    session.begin("Author geometry").unwrap();
    let report = session
        .create_inferred_sketch_line(sketch, &request())
        .unwrap();
    assert_eq!(report.point_constraints.len(), 1);
    assert_eq!(report.direction.unwrap().geometric_index, 1);
    assert_numeric_verification(&session, sketch, 1);
    assert_centerline(&session, sketch, report.primitive.entity);
    session.end().unwrap();
    let label = session.undo_labels().last().copied().unwrap();
    assert_eq!(label, "Author geometry");
    session.undo().unwrap();
    assert!(geometry(&session, sketch).lines().is_empty());
    session.redo().unwrap();
    assert_numeric_verification(&session, sketch, 1);
}
#[test]
fn rejected_reply_discards_line_and_relations_but_reserves_all_their_ids() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let before = session.snapshot().sketches.clone();
    let result = session.create_inferred_sketch_line_projected(sketch, &request(), |result| {
        let report = result.unwrap();
        assert_eq!(report.primitive.entity, SketchEntityId(4));
        assert_eq!(report.direction.unwrap().constraint.0, 2);
        Err::<(), _>("reply refused")
    });
    assert!(result.unwrap_err().to_string().contains("reply refused"));
    assert_eq!(session.snapshot().sketches, before);
    let report = session
        .create_inferred_sketch_line(sketch, &request())
        .unwrap();
    assert_eq!(report.primitive.entity, SketchEntityId(7));
    assert_eq!(report.point_constraints[0].0, 3);
    assert_eq!(report.direction.unwrap().constraint.0, 4);
}
#[test]
fn commit_veto_keeps_no_geometry_and_the_next_attempt_uses_new_ids() {
    let mut session = prepared_session(Some(EVENT_TRANSACTION_COMMITTED), 0);
    let sketch = DocumentSketchId(1);
    assert!(
        session
            .create_inferred_sketch_line(sketch, &request())
            .is_err()
    );
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert_next_vetoed_ids(&mut session, sketch);
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert!(session.undo_labels().is_empty());
}
#[test]
fn document_working_scale_sets_the_snap_reach() {
    for (scale, expected_snaps) in [(0.1, 1), (10.0, 0)] {
        let mut session = scaled_session(scale);
        session
            .create_sketch_primitive(DocumentSketchId(1), &point([0.0, 0.0]))
            .unwrap();
        let mut input = request();
        input.endpoints[0] = [0.0005, 0.0005];
        let report = session
            .create_inferred_sketch_line(DocumentSketchId(1), &input)
            .unwrap();
        assert_eq!(report.point_constraints.len(), expected_snaps);
    }
}

fn assert_centerline(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) {
    assert!(geometry(session, sketch).is_centerline(line).unwrap());
}

fn assert_next_vetoed_ids(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
) {
    assert!(
        session
            .create_inferred_sketch_line_projected(sketch, &request(), |result| {
                let report = result.unwrap();
                assert_eq!(report.primitive.entity, SketchEntityId(6));
                assert_eq!(report.direction.unwrap().constraint.0, 2);
                Ok::<_, String>(report)
            })
            .is_err()
    );
}

fn scaled_session(scale: f64) -> DocumentSession<RecordingObserver> {
    let mut snapshot = prepared_snapshot();
    snapshot.parameters =
        ParameterCollection::new(UnitsOfMeasure::default().with_working_scale(scale).unwrap());
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
