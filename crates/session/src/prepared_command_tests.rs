// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::part_evaluation::PartEvaluationSlot;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::FeatureId,
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
};

struct VetoObserver(bool);
impl TransactionObserver for VetoObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        if self.0 {
            return Err("injected veto".into());
        }
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}

fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}

fn retained_session(
    veto: bool,
) -> (
    DocumentSession<VetoObserver>,
    PartEvaluationSlot<NamedSolid, EvaluationFailure>,
) {
    let snapshot = super::fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut session = DocumentSession::new(
        7,
        DocumentEditHistory::new(snapshot, ids).unwrap(),
        VetoObserver(veto),
    );
    let result = {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(&session.history, document(), None);
        session
            .prepare_part_evaluation()
            .evaluate(&mut evaluator)
            .unwrap()
    };
    let mut slot = PartEvaluationSlot::default();
    session.adopt_part_evaluation(result, &mut slot).unwrap();
    (session, slot)
}

#[test]
fn veto_leaves_generation_and_retained_body_current() {
    let (mut session, slot) = retained_session(true);
    let generation = session.generation();
    let result = session.run_prepared(
        "depth",
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(3, "3 cm")
                .map_err(|error| error.to_string())
        },
        |_, ()| Ok(()),
        |_, (), _| panic!("veto must discard publication"),
    );
    assert!(matches!(
        result,
        Err(CommandError::Session(SessionError::Veto(_)))
    ));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    let volume = slot
        .current(&session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .volume(&session.history, document())
        .unwrap();
    assert!((volume - 24.0).abs() < 1e-8);
}

#[test]
fn preflight_failure_leaves_generation_and_retained_body_current() {
    let (mut session, slot) = retained_session(false);
    let generation = session.generation();
    let result = session.run_prepared(
        "depth",
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(3, "3 cm")
                .map_err(|error| error.to_string())
        },
        |_, ()| Err::<(), _>("injected preflight failure".into()),
        |_, (), _| panic!("failed preflight must discard publication"),
    );
    assert!(result.is_err());
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    let volume = slot
        .current(&session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .volume(&session.history, document())
        .unwrap();
    assert!((volume - 24.0).abs() < 1e-8);
}

#[test]
fn prepared_success_publishes_one_undo_step() {
    let (mut session, slot) = retained_session(false);
    let mut published = false;
    session
        .run_prepared(
            "depth",
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(3, "3 cm")
                    .map_err(|error| error.to_string())
            },
            |_, ()| Ok(()),
            |_, (), _| published = true,
        )
        .unwrap();
    assert!(published);
    assert_eq!(session.undo_labels(), ["depth"]);
    assert!(slot.current(&session).is_err());
    session.undo().unwrap();
    assert_eq!(
        session
            .snapshot()
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
}

#[test]
fn veto_keeps_real_body_and_prepared_dependent_unchanged() {
    let (mut session, slot) = retained_session(true);
    let source_generation = session.generation();
    let snapshot = super::fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut dependent = DocumentEditHistory::new(snapshot, ids).unwrap();
    let dependent_generation = dependent.generation();
    let result = session.run_prepared(
        "depth",
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(3, "3 cm")
                .map_err(|error| error.to_string())
        },
        |edit, ()| {
            let staged = dependent
                .prepare_refresh(|snapshot| snapshot.parameters.set_expression(3, "4 cm"))
                .map_err(|error| error.to_string())?;
            Ok(oblikovati_model::history::PreparedDependentRefresh::new(
                edit, staged,
            ))
        },
        |_, _, _| panic!("veto must not publish the dependent"),
    );
    assert!(matches!(
        result,
        Err(CommandError::Session(SessionError::Veto(_)))
    ));
    assert_eq!(session.generation(), source_generation);
    assert_eq!(dependent.generation(), dependent_generation);
    let volume = slot
        .current(&session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .volume(&session.history, document())
        .unwrap();
    assert!((volume - 24.0).abs() < 1e-8);
    assert_eq!(
        dependent
            .snapshot()
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
}

fn owned_session(veto: bool) -> DocumentSession<VetoObserver> {
    let (mut session, _) = retained_session(veto);
    let completed = {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(&session.history, document(), None);
        session
            .prepare_part_evaluation()
            .evaluate_checkpoints(&mut evaluator)
            .unwrap()
    };
    session
        .adopt_owned_full_checkpoints(None, completed)
        .unwrap();
    session
}

#[test]
fn prepared_real_replay_and_dependent_publish_in_one_session_command() {
    let mut session = owned_session(false);
    let snapshot = super::fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut dependent = DocumentEditHistory::new(snapshot, ids).unwrap();
    session.validate_prepared_named_destination(None).unwrap();
    session
        .run_prepared(
            "depth",
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(3, "3 cm")
                    .map_err(|error| error.to_string())
            },
            |edit, ()| {
                let bodies =
                    oblikovati_martensite::prepare_full_named_publication(edit, document(), None)?;
                let refresh = dependent
                    .prepare_refresh(|snapshot| snapshot.parameters.set_expression(3, "4 cm"))
                    .map_err(|error| error.to_string())?;
                Ok((
                    bodies,
                    oblikovati_model::history::PreparedDependentRefresh::new(edit, refresh),
                ))
            },
            |session, (bodies, _), _| session.publish_prepared_named_replay(None, bodies),
        )
        .unwrap();
    let body = session
        .owned_part_slot(None)
        .unwrap()
        .current(&session)
        .unwrap();
    let volume = body
        .body(FeatureId(1))
        .unwrap()
        .volume(&session.history, document())
        .unwrap();
    assert!((volume - 36.0).abs() < 1e-8);
    assert_eq!(
        dependent
            .snapshot()
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        4.0
    );
    assert_eq!(session.undo_labels(), ["depth"]);
}

#[test]
fn body_preflight_refusal_keeps_owned_solid_and_dependent_current() {
    let mut session = owned_session(false);
    let source_generation = session.generation();
    let snapshot = super::fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut dependent = DocumentEditHistory::new(snapshot, ids).unwrap();
    let dependent_generation = dependent.generation();
    session.validate_prepared_named_destination(None).unwrap();
    let result = session.run_prepared(
        "depth",
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(3, "3 cm")
                .map_err(|error| error.to_string())
        },
        |edit, ()| {
            let _body =
                oblikovati_martensite::prepare_full_named_publication(edit, document(), None)?;
            let _dependent = dependent
                .prepare_refresh(|snapshot| snapshot.parameters.set_expression(3, "4 cm"))
                .map_err(|error| error.to_string())?;
            Err::<(), _>("injected body destination refusal".into())
        },
        |_, (), _| panic!("refused body must not publish"),
    );
    assert!(result.is_err());
    assert_eq!(session.generation(), source_generation);
    assert_eq!(dependent.generation(), dependent_generation);
    let part = session
        .owned_part_slot(None)
        .unwrap()
        .current(&session)
        .unwrap();
    let volume = part
        .body(FeatureId(1))
        .unwrap()
        .volume(&session.history, document())
        .unwrap();
    assert!((volume - 24.0).abs() < 1e-8);
    assert!(session.undo_labels().is_empty());
}
