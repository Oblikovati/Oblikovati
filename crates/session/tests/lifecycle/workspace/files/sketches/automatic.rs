// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::{document_sketch::SketchReadinessCause, history::DocumentEditHistory};
pub(super) fn stale_session(
    options: SolveOptions,
) -> (DocumentSession<RecordingObserver>, DocumentSketchId) {
    let (host, _, id, sketch) = saved_driven_sketch();
    let mut snapshot = host.workspace().session(id).unwrap().snapshot().clone();
    snapshot.parameters.set_expression(1, "6 cm").unwrap();
    snapshot
        .properties
        .sync_exposed_parameters(&snapshot.parameters)
        .unwrap();
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    (
        DocumentSession::new_with_sketch_options(
            17,
            history,
            RecordingObserver::default(),
            options,
        ),
        sketch,
    )
}
fn readiness(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
) -> Option<SketchReadinessCause> {
    let snapshot = session.snapshot();
    snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .require_fully_constrained(&snapshot.parameters)
        .err()
        .map(|error| error.cause)
}
#[test]
fn explicit_cancellation_does_not_trigger_an_automatic_retry_of_stale_geometry() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let report = session
        .recompute_sketches(Default::default(), || true)
        .unwrap();
    assert!(report.cancelled);
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(
        readiness(&session, sketch),
        Some(SketchReadinessCause::Stale(vec![1]))
    );
    assert!(session.undo_labels().is_empty());
}
#[test]
fn automatic_recompute_uses_injected_policy_and_retains_unconverged_geometry() {
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let (mut session, sketch) = stale_session(options);
    let before = session.snapshot().sketches.clone();
    session.run("Inspect", |_, _| Ok::<(), String>(())).unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert_eq!(
        readiness(&session, sketch),
        Some(SketchReadinessCause::Unconverged(
            SolveTermination::IterationLimit
        ))
    );
    session
        .recompute_sketches(Default::default(), || false)
        .unwrap();
    assert_eq!(readiness(&session, sketch), None);
}
#[test]
fn failed_edits_discard_parameter_geometry_and_derived_health_together() {
    let (mut host, _, id, _) = saved_driven_sketch();
    let before = sketch_state(&host, id).clone();
    let result = host
        .commands(id)
        .unwrap()
        .run::<(), String>("Fail", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm").unwrap();
            Err("declined".into())
        });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(sketch_state(&host, id)));
}
#[test]
fn group_abort_discards_automatic_geometry_and_derived_health() {
    let (mut host, _, id, _) = saved_driven_sketch();
    let before = sketch_state(&host, id).clone();
    host.commands(id).unwrap().begin("Abort geometry").unwrap();
    host.commands(id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    host.commands(id).unwrap().abort().unwrap();
    assert!(before.same_evaluated_state(sketch_state(&host, id)));
}

fn sketch_state(host: &Host, id: u64) -> &oblikovati_model::document_sketch::DocumentSketches {
    &host.workspace().session(id).unwrap().snapshot().sketches
}
#[test]
fn ordinary_parameter_edit_saves_and_reopens_the_automatic_geometry_without_an_explicit_solve() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    host.commands(id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "9 cm")
        })
        .unwrap();
    assert!((point_height(&host, id, sketch) - 9.0).abs() < 1e-8);
    host.save(id, "alpha", &mut source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!((point_height(&host, reopened, sketch) - 9.0).abs() < 1e-8);
}
