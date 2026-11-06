// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId},
};

fn visibility_session() -> DocumentSession<RecordingObserver> {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [2.0, 3.0]).unwrap();
    sketch_fixture::owned_sketch_session(None, "Display", geometry)
}

#[test]
fn visibility_metadata_has_one_history_step_and_no_numerical_side_effect() {
    let mut session = visibility_session();
    let original = session.snapshot().sketches.clone();
    session
        .set_sketch_visibility(DocumentSketchId(1), false)
        .unwrap();
    assert!(
        !session
            .snapshot()
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .visible()
    );
    session.undo().unwrap();
    assert!(original.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    let revision = session.generation();
    session
        .set_sketch_visibility(DocumentSketchId(1), false)
        .unwrap();
    assert_eq!(session.generation(), revision);
}

#[test]
fn rejected_visibility_reply_retains_geometry_verification_and_display_state() {
    let mut session = visibility_session();
    let original = session.snapshot().sketches.clone();
    let revision = session.generation();
    let result = session.set_sketch_visibility_projected(
        DocumentSketchId(1),
        false,
        |reply| -> Result<(), String> {
            assert!(
                !reply
                    .unwrap()
                    .sketches
                    .get(DocumentSketchId(1))
                    .unwrap()
                    .visible()
            );
            Err("visibility reply rejected".into())
        },
    );
    assert!(result.is_err());
    assert!(original.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), revision);
    let cause = session
        .set_sketch_visibility(DocumentSketchId(999), false)
        .unwrap_err();
    assert!(cause.to_string().contains("999"));
    assert!(original.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn rejected_changes_and_noop_replies_keep_the_callers_edit_group() {
    let mut session = visibility_session();
    session.begin("Grouped visibility").unwrap();
    let original = session.snapshot().sketches.clone();
    let revision = session.generation();
    for visible in [false, true] {
        let result = session.set_sketch_visibility_projected(
            DocumentSketchId(1),
            visible,
            |reply| -> Result<(), String> {
                assert_eq!(
                    reply
                        .unwrap()
                        .sketches
                        .get(DocumentSketchId(1))
                        .unwrap()
                        .visible(),
                    visible
                );
                Err("grouped reply rejected".into())
            },
        );
        assert!(result.is_err());
        assert!(session.in_transaction());
        assert_eq!(session.generation(), revision);
        assert!(original.same_evaluated_state(&session.snapshot().sketches));
    }
    session.end().unwrap();
    assert!(session.undo_labels().is_empty());
}
