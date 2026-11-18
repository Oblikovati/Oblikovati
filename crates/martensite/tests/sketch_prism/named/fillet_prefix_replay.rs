// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    AuthoredFeature, fillet::FilletDefinition, part::PartFeatureDefinition,
};
use oblikovati_session::{
    DocumentSession, TransactionObserver, part_evaluation::PartEvaluationSlot,
};

struct CaptureObserver;
impl TransactionObserver for CaptureObserver {
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

#[test]
fn transaction_captures_a_named_fillet_before_retiring_the_prefix() {
    let mut session = DocumentSession::new(7, owner(cube_document()), CaptureObserver);
    let edge = certified(session.history())
        .subentity_reference_keys(session.history(), document())
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    let definition =
        FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(0.1)).unwrap();
    let mut slot = PartEvaluationSlot::default();
    let completed = {
        let mut backend = CertifiedNamedSolidEvaluator::new(
            session.history(),
            document(),
            certified_resolution(),
        );
        session
            .prepare_part_evaluation()
            .evaluate(&mut backend)
            .unwrap()
    };
    session.adopt_part_evaluation(completed, &mut slot).unwrap();
    let (captured, transition) = {
        let prefix = slot.candidate_prefix(&session).unwrap();
        session
            .run_projected_with_transition(
                "Fillet",
                |snapshot, ids| {
                    snapshot
                        .create_fillet("Fillet", definition, ids)
                        .map_err(|error| error.to_string())
                },
                |snapshot, feature, history| {
                    let mut backend = CertifiedNamedSolidEvaluator::new(
                        history,
                        document(),
                        certified_resolution(),
                    );
                    let candidate = prefix
                        .prepare_tail(history, snapshot, feature, &mut backend)?
                        .ok_or("expected eligible prefix")?;
                    candidate.capture(history, |_, body| body.capture_current(history, document()))
                },
            )
            .unwrap()
    };
    slot.clear();
    assert_eq!(captured.body_count(), 1);
    assert!(captured.matches_transition(&transition, session.history()));
    session.undo().unwrap();
    assert!(!captured.matches_transition(&transition, session.history()));
    let closing_scope = session.history().evaluation_scope();
    session.close();
    assert!(closing_scope.is_superseded());
}

#[test]
fn retained_cube_prefix_builds_the_same_named_fillet_as_full_replay() {
    let mut history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let retained = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let prism = retained.body(FeatureId(1)).unwrap();
    let edge = prism
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    let definition =
        FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(0.1)).unwrap();
    let mut candidate = history.snapshot().clone();
    let reads = definition.reads();
    candidate
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    let tail = retained
        .replay_candidate_tail(&candidate, &mut backend)
        .unwrap();
    let full = candidate.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(tail.health(), full.health());
    let actual = tail.body(FeatureId(2)).unwrap();
    let expected = full.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual.reference_keys(&history, document()).unwrap(),
        expected.reference_keys(&history, document()).unwrap()
    );
    assert!(
        (actual.volume(&history, document()).unwrap()
            - expected.volume(&history, document()).unwrap())
        .abs()
            < 1e-10
    );
    let body_key = actual.body_reference_key(&history, document()).unwrap();
    assert_eq!(
        actual
            .topology_summary(&history, document(), body_key)
            .unwrap(),
        expected
            .topology_summary(&history, document(), body_key)
            .unwrap()
    );
    for face in actual.face_reference_keys(&history, document()).unwrap() {
        let left = actual.face_area(&history, document(), &face).unwrap();
        let right = expected.face_area(&history, document(), &face).unwrap();
        assert!(
            (left - right).abs() < 1e-10,
            "face {face:?}: {left} != {right}"
        );
    }
    assert!(retained.body(FeatureId(1)).is_some());
    let copied_prism = retained
        .body(FeatureId(1))
        .unwrap()
        .capture_current(&history, document())
        .unwrap();
    let copied_fillet = actual.capture_current(&history, document()).unwrap();
    assert_eq!(
        copied_fillet.reference_keys(&history, document()).unwrap(),
        actual.reference_keys(&history, document()).unwrap()
    );
    assert_eq!(
        copied_prism.reference_keys(&history, document()).unwrap(),
        retained
            .body(FeatureId(1))
            .unwrap()
            .reference_keys(&history, document())
            .unwrap()
    );
    drop(tail);
    drop(full);
    drop(retained);
    drop(backend);
    assert!(copied_prism.volume(&history, document()).unwrap() > 0.);
    assert!(copied_fillet.volume(&history, document()).unwrap() > 0.);
    history
        .refresh(|snapshot| snapshot.parameters.set_expression(1, "20 mm"))
        .unwrap();
    assert!(copied_prism.volume(&history, document()).is_err());
    assert!(copied_fillet.volume(&history, document()).is_err());
    assert!(copied_fillet.capture_current(&history, document()).is_err());
}
