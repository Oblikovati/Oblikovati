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
type NativePartSlot =
    PartEvaluationSlot<PrismProbeSnapshot, oblikovati_martensite::EvaluationFailure>;
#[test]
fn session_created_extrusion_builds_exact_faces_through_parameter_history() {
    let mut snapshot = selected_prism_document();
    snapshot.work_geometry = Default::default();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let mut owner = DocumentSession::new(7, history, NativeResultObserver);
    let id = owner
        .create_sketch_prism("Extrude", stored_definition())
        .unwrap();
    let mut slot = NativePartSlot::default();
    assert_created_faces(&mut owner, &mut slot, id, [6., 6., 8., 8., 12., 12.]);
    native_depth_edit(&mut owner);
    assert_created_faces(&mut owner, &mut slot, id, [9., 9., 12., 12., 12., 12.]);
    owner.undo().unwrap();
    assert_created_faces(&mut owner, &mut slot, id, [6., 6., 8., 8., 12., 12.]);
    owner.redo().unwrap();
    assert_created_faces(&mut owner, &mut slot, id, [9., 9., 12., 12., 12., 12.]);
}
fn assert_created_faces(
    owner: &mut NativePartSession,
    slot: &mut NativePartSlot,
    id: FeatureId,
    expected: [f64; 6],
) {
    publish_native(owner, slot);
    let body = slot.current(owner).unwrap().body(id).unwrap();
    let report = body.report();
    assert!(report.topology.solid);
    assert_eq!(report.topology.faces, 6);
    assert_eq!(report.topology.edges, 12);
    assert_eq!(report.topology.vertices, 8);
    assert_eq!(report.faces.len(), expected.len());
    assert_source_roles(report);
    assert!(report.faces.iter().all(|face| face.loops == 1));
    let mut areas: Vec<f64> = report.faces.iter().map(|face| face.area).collect();
    areas.sort_by(f64::total_cmp);
    for (area, expected_area) in areas.into_iter().zip(expected) {
        assert_volume(area, expected_area);
    }
}
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
