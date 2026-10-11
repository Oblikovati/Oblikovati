// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn moved_point(session: &DocumentSession<RecordingObserver>) -> Option<[f64; 2]> {
    session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .dimension_text_point(SketchConstraintId(1))
        .unwrap()
}
fn move_label(session: &mut DocumentSession<RecordingObserver>) {
    session
        .move_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), [2.0, 3.0])
        .unwrap();
}
#[test]
fn placement_preserves_numerical_verification_and_same_bits_are_a_history_noop() {
    let mut session = dimension_session(None);
    session.solve_sketch(DocumentSketchId(1), || false).unwrap();
    let before = session.snapshot().clone();
    move_label(&mut session);
    assert_retained_verification(&before, session.snapshot());
    let labels = session.undo_labels().len();
    move_label(&mut session);
    assert_eq!(session.undo_labels().len(), labels);
    assert_eq!(moved_point(&session), Some([2.0, 3.0]));
}
#[test]
fn annotation_history_and_groups_restore_exact_placement() {
    let mut session = dimension_session(None);
    move_label(&mut session);
    session.undo().unwrap();
    assert_eq!(moved_point(&session), None);
    session.redo().unwrap();
    assert_eq!(moved_point(&session), Some([2.0, 3.0]));
    session.begin("Annotation group").unwrap();
    session
        .move_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), [-0.0, 9.0])
        .unwrap();
    session.abort().unwrap();
    assert_eq!(moved_point(&session), Some([2.0, 3.0]));
}
#[test]
fn rejected_reply_and_veto_cannot_adopt_annotation_metadata() {
    let mut session = dimension_session(None);
    let result: Result<(), CommandError<String>> = session.move_sketch_dimension_projected(
        DocumentSketchId(1),
        SketchConstraintId(1),
        [2.0, 3.0],
        |reply| {
            reply.unwrap();
            Err("reply rejected".into())
        },
    );
    assert!(result.unwrap_err().to_string().contains("reply rejected"));
    assert_eq!(moved_point(&session), None);
}
#[test]
fn veto_cannot_adopt_annotation_metadata() {
    let mut vetoed = dimension_session(Some(EVENT_TRANSACTION_COMMITTED));
    assert!(
        vetoed
            .move_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), [2.0, 3.0])
            .is_err()
    );
    assert_eq!(moved_point(&vetoed), None);
}

fn assert_retained_verification(before: &DocumentSnapshot, after: &DocumentSnapshot) {
    let first = before.sketches.get(DocumentSketchId(1)).unwrap();
    let second = after.sketches.get(DocumentSketchId(1)).unwrap();
    assert_eq!(
        first.health(&before.parameters),
        second.health(&after.parameters)
    );
}
#[test]
fn failed_dimension_source_does_not_prevent_annotation_only_edit() {
    let mut session = dimension_session(None);
    session
        .run("Fail dimension", |snapshot, _| {
            snapshot.parameters.set_expression(2, "missing")
        })
        .unwrap();
    let before = session.snapshot().clone();
    move_label(&mut session);
    assert_retained_verification(&before, session.snapshot());
    assert_eq!(moved_point(&session), Some([2.0, 3.0]));
}
