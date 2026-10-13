// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    feature::{FeatureId, part::SketchPrismEvaluator, sketch_prism::ResolvedSketchPrism},
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
};
use oblikovati_session::{DocumentSession, TransactionObserver, part_evaluation::*};
#[derive(Default)]
struct RecordingPartObserver {
    changes: usize,
    events: usize,
}
impl TransactionObserver for RecordingPartObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        self.events += 1;
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {
        self.events += 1;
    }
    fn changed(&mut self, _: u64, _: u64) {
        self.changes += 1;
    }
}
#[derive(Default)]
struct FakePrismEvaluator {
    calls: usize,
}
impl SketchPrismEvaluator for FakePrismEvaluator {
    type Body = f64;
    type Failure = &'static str;
    fn evaluate_prism(
        &mut self,
        _feature: oblikovati_model::feature::FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<f64, &'static str> {
        self.calls += 1;
        Ok(input.displacement[2])
    }
}
type PartSession = DocumentSession<RecordingPartObserver>;
type PartSlot = PartEvaluationSlot<f64, &'static str>;
fn session(id: u64) -> PartSession {
    let document = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentSession::new(
        id,
        DocumentEditHistory::new(document, ids).unwrap(),
        RecordingPartObserver::default(),
    )
}
fn edit_depth(session: &mut PartSession, expression: &str) {
    session
        .run("Depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, expression)
        })
        .unwrap();
}
fn publish(session: &mut PartSession, slot: &mut PartSlot) {
    let completed = complete_fake(session);
    session.adopt_part_evaluation(completed, slot).unwrap();
}
fn current_depth(slot: &PartSlot, session: &PartSession) -> f64 {
    *slot.current(session).unwrap().body(FeatureId(1)).unwrap()
}
#[test]
fn adopting_current_results_keeps_recipe_history_and_events_unchanged() {
    let mut session = session(7);
    let mut slot = PartSlot::default();
    verify_empty_slot(&slot, &session);
    publish(&mut session, &mut slot);
    assert_eq!(current_depth(&slot, &session), 2.);
    let latest = slot.latest().unwrap();
    assert_eq!(
        (latest.document, latest.generation, latest.is_current),
        (7, 0, true)
    );
    assert!(session.undo_labels().is_empty());
    assert_eq!(
        (session.observer().events, session.observer().changes),
        (0, 0)
    );
    slot.clear();
    assert!(slot.latest().is_none());
}
#[test]
fn an_edit_marks_presentation_stale_and_newer_adoption_survives_late_results() {
    let mut session = session(7);
    let mut slot = PartSlot::default();
    publish(&mut session, &mut slot);
    let older = complete_fake(&session);
    edit_depth(&mut session, "3 cm");
    let stale = slot.latest().unwrap();
    assert!(!stale.is_current);
    assert_eq!(*stale.evaluation.body(FeatureId(1)).unwrap(), 2.);
    assert!(matches!(
        slot.current(&session),
        Err(PartEvaluationError::Superseded { .. })
    ));
    publish(&mut session, &mut slot);
    let error = session.adopt_part_evaluation(older, &mut slot).unwrap_err();
    assert!(error.to_string().contains("expected current generation"));
    assert_eq!(current_depth(&slot, &session), 3.);
    assert_eq!(session.undo_labels(), ["Depth"]);
}
#[test]
fn foreign_document_and_reopened_history_cannot_replace_current_output() {
    let mut session = session(7);
    let mut slot = PartSlot::default();
    publish(&mut session, &mut slot);
    for id in [8, 7] {
        let completed = complete_fake(&self::session(id));
        let error = session
            .adopt_part_evaluation(completed, &mut slot)
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        if id == 8 {
            assert!(matches!(error, PartEvaluationError::ForeignDocument { .. }));
        } else {
            assert!(matches!(error, PartEvaluationError::ForeignHistory { .. }));
        }
        assert_eq!(current_depth(&slot, &session), 2.);
    }
}
#[test]
fn close_cancels_a_queued_request_without_a_backend_call() {
    let session = session(7);
    let request = session.prepare_part_evaluation();
    assert!(!request.is_superseded());
    session.close();
    assert!(request.is_superseded());
    let mut backend = FakePrismEvaluator::default();
    let error = request.evaluate(&mut backend).err().unwrap();
    assert!(error.to_string().contains("cancelled after 0"));
    assert!(
        std::error::Error::source(&error)
            .unwrap()
            .source()
            .is_none()
    );
    assert_eq!(backend.calls, 0);
}
#[test]
fn undo_redo_and_abort_invalidate_queued_generations() {
    let mut session = session(7);
    edit_depth(&mut session, "3 cm");
    let before_undo = session.prepare_part_evaluation();
    session.undo().unwrap();
    assert!(before_undo.is_superseded());
    let before_redo = session.prepare_part_evaluation();
    session.redo().unwrap();
    assert!(before_redo.is_superseded());
    session.begin("Preview").unwrap();
    session
        .edit(|snapshot, _| snapshot.parameters.set_expression(3, "4 cm"))
        .unwrap();
    let preview = session.prepare_part_evaluation();
    session.abort().unwrap();
    assert!(preview.is_superseded());
}
struct EditingPrismEvaluator<'a> {
    owner: &'a mut PartSession,
    calls: usize,
}
impl SketchPrismEvaluator for EditingPrismEvaluator<'_> {
    type Body = f64;
    type Failure = &'static str;
    fn evaluate_prism(
        &mut self,
        _feature: oblikovati_model::feature::FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<f64, &'static str> {
        self.calls += 1;
        edit_depth(self.owner, "3 cm");
        Ok(input.displacement[2])
    }
}
#[test]
fn edit_during_evaluation_discards_a_partial_result() {
    let mut session = session(7);
    let request = session.prepare_part_evaluation();
    let mut backend = EditingPrismEvaluator {
        owner: &mut session,
        calls: 0,
    };
    let error = request.evaluate(&mut backend).err().unwrap();
    assert!(error.to_string().contains("cancelled after 1"));
    assert_eq!(backend.calls, 1);
    assert_eq!(session.undo_labels(), ["Depth"]);
}

#[test]
fn a_foreign_destination_slot_requires_explicit_retirement() {
    let mut first = session(7);
    let mut second = session(8);
    let mut slot = PartSlot::default();
    publish(&mut first, &mut slot);
    let completed = second
        .prepare_part_evaluation()
        .evaluate(&mut FakePrismEvaluator::default())
        .unwrap();
    assert!(matches!(
        second.adopt_part_evaluation(completed, &mut slot),
        Err(PartEvaluationError::ForeignDocument { .. })
    ));
    assert_eq!(current_depth(&slot, &first), 2.);
    slot.clear();
    publish(&mut second, &mut slot);
    assert_eq!(slot.latest().unwrap().document, 8);
}

fn complete_fake(session: &PartSession) -> CompletedPartEvaluation<f64, &'static str> {
    session
        .prepare_part_evaluation()
        .evaluate(&mut FakePrismEvaluator::default())
        .unwrap()
}
fn verify_empty_slot(slot: &PartSlot, session: &PartSession) {
    let error = slot.current(session).err().unwrap();
    assert!(error.to_string().contains("document 7"));
    assert!(std::error::Error::source(&error).is_none());
}
