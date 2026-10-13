// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityEditError, SketchEntityId,
        SketchLineEnd, SketchLineEndpointReport, SketchPlane,
    },
};
fn endpoint_session(veto: Option<&'static str>) -> DocumentSession<RecordingObserver> {
    let mut snapshot = recipe();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Endpoint",
            SketchPlane::xy(),
            endpoint_geometry(),
        )
        .unwrap();
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    DocumentSession::new(
        17,
        history,
        RecordingObserver {
            veto,
            ..Default::default()
        },
    )
}
fn endpoint_geometry() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [2.0, 0.0])] {
        sketch
            .add_defining_point(SketchEntityId(id), position)
            .unwrap();
        sketch
            .add_constraint(
                SketchConstraintId(id),
                PlanarConstraint::Fix {
                    point: SketchEntityId(id),
                    position,
                },
            )
            .unwrap();
    }
    sketch
        .add_line(SketchEntityId(3), [1, 2].map(SketchEntityId))
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(3),
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(3),
            },
        )
        .unwrap();
    sketch
}
fn replace_endpoint(
    session: &mut DocumentSession<RecordingObserver>,
) -> Result<SketchLineEndpointReport, CommandError<SketchEntityEditError>> {
    session.run("Replace endpoint", |snapshot, ids| {
        snapshot.replace_sketch_line_endpoint(
            DocumentSketchId(1),
            SketchEntityId(3),
            SketchLineEnd::Last,
            [5.0, 7.0],
            ids,
        )
    })
}
#[test]
fn endpoint_replacement_recomputes_constraints_and_undo_redo_restore_exact_states() {
    let mut session = endpoint_session(None);
    session
        .recompute_sketches(Default::default(), || false)
        .unwrap();
    let before = session.snapshot().sketches.clone();
    let report = replace_endpoint(&mut session).unwrap();
    let after = session.snapshot().sketches.clone();
    assert_eq!((report.previous.0, report.created.0), (2, 4));
    assert_solved_endpoint(&session, report.created);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.history().sketch_reservations().entity, 4);
}
fn assert_solved_endpoint(session: &DocumentSession<RecordingObserver>, created: SketchEntityId) {
    let sketch = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    // The fixture's largest seed coordinate is seven working units.
    let limit = oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance * 7.0;
    for (id, expected) in [(created, [5.0, 0.0]), (SketchEntityId(2), [2.0, 0.0])] {
        let position = sketch.point(id).unwrap().position;
        for axis in 0..2 {
            assert!(
                (position[axis] - expected[axis]).abs() <= limit,
                "{id:?}: {position:?}"
            );
        }
    }
    assert_eq!(sketch.constraints(), endpoint_geometry().constraints());
}
#[test]
fn endpoint_replacement_veto_and_group_abort_keep_old_geometry_and_burn_ids() {
    for veto in [None, Some(EVENT_TRANSACTION_COMMITTED)] {
        let mut session = endpoint_session(veto);
        let before = session.snapshot().sketches.clone();
        if veto.is_none() {
            session.begin("Abort endpoint").unwrap();
        }
        let result = replace_endpoint(&mut session);
        assert_eq!(result.is_err(), veto.is_some());
        if veto.is_none() {
            session.abort().unwrap();
        }
        assert!(before.same_evaluated_state(&session.snapshot().sketches));
        assert_eq!(session.history().sketch_reservations().entity, 4);
        assert!(session.undo_labels().is_empty());
    }
}
#[test]
fn endpoint_reply_rejection_discards_candidate_before_solver_and_retains_reservation() {
    let mut session = endpoint_session(None);
    let before = session.snapshot().sketches.clone();
    let result = session.run("Rejected endpoint", |snapshot, ids| {
        snapshot
            .replace_sketch_line_endpoint(
                DocumentSketchId(1),
                SketchEntityId(3),
                SketchLineEnd::Last,
                [5.0, 7.0],
                ids,
            )
            .unwrap();
        Err::<(), _>("reply rejected")
    });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.history().sketch_reservations().entity, 4);
    assert_eq!(replace_endpoint(&mut session).unwrap().created.0, 5);
}
#[test]
fn endpoint_replacement_rejects_missing_document_owners_without_reserving() {
    let mut session = endpoint_session(None);
    let before = session.snapshot().sketches.clone();
    let result = session.run("Missing endpoint owner", |snapshot, ids| {
        snapshot.replace_sketch_line_endpoint(
            DocumentSketchId(99),
            SketchEntityId(3),
            SketchLineEnd::Last,
            [5.0, 0.0],
            ids,
        )
    });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.history().sketch_reservations().entity, 3);
}
