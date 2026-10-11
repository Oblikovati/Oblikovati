// SPDX-License-Identifier: GPL-2.0-only
use super::automatic::stale_session;
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::document_sketch::DocumentSketches;
use oblikovati_model::{history::DocumentEditHistory, sketch::SketchEntityId};

#[test]
fn selected_command_uses_injected_policy_without_an_automatic_retry() {
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let (mut session, sketch) = stale_session(options);
    let before = session.snapshot().sketches.clone();
    let report = session.solve_sketch(sketch, || false).unwrap();
    let outcome = report.reports[0].result.as_ref().unwrap();
    assert_eq!(
        outcome.numerical().termination,
        SolveTermination::IterationLimit
    );
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(!before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn selected_cancellation_and_missing_identity_leave_history_and_geometry_unchanged() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let report = session.solve_sketch(sketch, || true).unwrap();
    assert!(report.cancelled);
    assert!(
        session
            .solve_sketch(DocumentSketchId(99), || false)
            .is_err()
    );
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
    assert!(!session.in_transaction());
}

#[test]
fn selected_command_joins_history_and_restores_geometry_on_undo_redo() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    session.begin("Verify selected sketch").unwrap();
    session.solve_sketch(sketch, || false).unwrap();
    session.end().unwrap();
    let solved = session.snapshot().sketches.clone();
    assert!(!before.same_recipe(&solved));
    assert_eq!(session.undo_labels(), ["Verify selected sketch"]);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(solved.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn workspace_selected_command_preserves_saved_state_and_reopens_with_identity() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    let before = owner_sketches(&host, id).clone();
    let mut commands = host.commands(id).unwrap();
    let report = commands.solve_sketch(sketch, || false).unwrap();
    assert_eq!(report.reports.len(), 1);
    assert_eq!(report.reports[0].sketch, sketch);
    assert!(!host.workspace().needs_save(id).unwrap());
    host.save(id, "alpha", &mut source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!(before.same_evaluated_state(owner_sketches(&host, reopened)));
}

fn owner_sketches(host: &Host, document: u64) -> &DocumentSketches {
    &host
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
}

fn session_with_unverified_neighbor() -> (DocumentSession<RecordingObserver>, DocumentSketchId) {
    let (session, sketch) = stale_session(Default::default());
    let mut snapshot = session.snapshot().clone();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(999), [2.0, 3.0]).unwrap();
    let neighbor = DocumentSketchId(99);
    snapshot
        .sketches
        .add(neighbor, "Unverified", SketchPlane::xy(), geometry)
        .unwrap();
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    (
        DocumentSession::new(17, history, RecordingObserver::default()),
        sketch,
    )
}

#[test]
fn selected_command_does_not_verify_an_unrelated_pending_sketch() {
    let (mut session, sketch) = session_with_unverified_neighbor();
    let neighbor = DocumentSketchId(99);
    let before = session.snapshot().sketches.get(neighbor).unwrap().clone();
    let report = session.solve_sketch(sketch, || false).unwrap();
    assert_eq!(report.reports.len(), 1);
    assert_eq!(&before, session.snapshot().sketches.get(neighbor).unwrap());
}

#[test]
fn group_abort_discards_selected_geometry_and_verification() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    session.begin("Abort selected solve").unwrap();
    session.solve_sketch(sketch, || false).unwrap();
    session.abort().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn rejected_response_discards_solved_geometry_and_verification_before_commit() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let result = session.solve_sketch_projected(
        sketch,
        || false,
        |report| {
            assert!(report.unwrap().reports[0].result.is_ok());
            Err::<(), _>("reply refused")
        },
    );
    assert!(result.unwrap_err().to_string().contains("reply refused"));
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn projection_can_inspect_missing_identity_without_adopting_a_candidate() {
    let (mut session, _) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let result = session
        .solve_sketch_projected(
            DocumentSketchId(99),
            || false,
            |report| Ok::<_, String>(report.unwrap_err().to_string()),
        )
        .unwrap();
    assert!(result.contains("99"));
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn owned_structure_analysis_does_not_solve_stale_geometry_or_change_history() {
    let (session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let structure = session.analyze_sketch(sketch).unwrap();
    assert_eq!(structure.degrees_of_freedom, 0);
    assert!(session.analyze_sketch(DocumentSketchId(99)).is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}
