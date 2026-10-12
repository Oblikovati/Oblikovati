// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{MartensitePrismEvaluator, PrismProbeSnapshot};
use oblikovati_session::{
    DocumentSession, TransactionObserver,
    part_evaluation::{PartEvaluationError, PartEvaluationSlot},
};
#[derive(Default)]
struct NativeResultObserver;
impl TransactionObserver for NativeResultObserver {
    fn before(
        &mut self,
        _: &oblikovati_api::wire::TransactionEventPayload,
        _: &DocumentSnapshot,
    ) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &oblikovati_api::wire::TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
type NativePartSession = DocumentSession<NativeResultObserver>;
type NativePartSlot = PartEvaluationSlot<PrismProbeSnapshot, String>;
fn native_part_session() -> NativePartSession {
    let document = selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentSession::new(
        7,
        DocumentEditHistory::new(document, ids).unwrap(),
        NativeResultObserver,
    )
}
fn publish_native(owner: &mut NativePartSession, slot: &mut NativePartSlot) {
    let completed = owner
        .prepare_part_evaluation()
        .evaluate(&mut MartensitePrismEvaluator)
        .unwrap();
    owner.adopt_part_evaluation(completed, slot).unwrap();
}
fn assert_native_volume(slot: &NativePartSlot, owner: &NativePartSession, expected: f64) {
    let evaluated = slot.current(owner).unwrap();
    assert_volume(
        evaluated
            .body(FeatureId(1))
            .unwrap()
            .report()
            .properties
            .volume,
        expected,
    );
}
fn native_depth_edit(owner: &mut NativePartSession) {
    owner
        .run("Depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
}
#[test]
fn retained_current_bodies_reject_late_results_and_track_undo_redo_generations() {
    let mut owner = native_part_session();
    let mut slot = NativePartSlot::default();
    publish_native(&mut owner, &mut slot);
    assert_native_volume(&slot, &owner, 24.);
    let older = owner
        .prepare_part_evaluation()
        .evaluate(&mut MartensitePrismEvaluator)
        .unwrap();
    native_depth_edit(&mut owner);
    assert!(!slot.latest().unwrap().is_current);
    publish_native(&mut owner, &mut slot);
    assert_native_volume(&slot, &owner, 36.);
    assert!(matches!(
        owner.adopt_part_evaluation(older, &mut slot),
        Err(PartEvaluationError::Superseded { .. })
    ));
    assert_native_volume(&slot, &owner, 36.);
    verify_native_navigation(&mut owner, &mut slot);
}
fn verify_native_navigation(owner: &mut NativePartSession, slot: &mut NativePartSlot) {
    owner.undo().unwrap();
    assert!(slot.current(owner).is_err());
    publish_native(owner, slot);
    assert_native_volume(slot, owner, 24.);
    owner.redo().unwrap();
    assert!(slot.current(owner).is_err());
    publish_native(owner, slot);
    assert_native_volume(slot, owner, 36.);
}
#[test]
fn closed_native_owner_marks_retained_presentation_stale_and_cancels_queued_input() {
    let mut owner = native_part_session();
    let mut slot = NativePartSlot::default();
    publish_native(&mut owner, &mut slot);
    let queued = owner.prepare_part_evaluation();
    owner.close();
    let retained = slot.latest().unwrap();
    assert!(!retained.is_current);
    let body = retained.evaluation.body(FeatureId(1)).unwrap();
    assert_volume(body.report().properties.volume, 24.);
    assert!(queued.is_superseded());
    assert!(queued.evaluate(&mut MartensitePrismEvaluator).is_err());
    slot.clear();
}
