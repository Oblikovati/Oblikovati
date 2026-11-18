// SPDX-License-Identifier: GPL-2.0-only
//! Exact named bodies survive a clean checkpoint within one history generation.
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid};
use oblikovati_model::feature::{
    fillet::FilletDefinition,
    part::{PartBlendKind, PartFeatureEvaluator, PartFeatureFailure},
    sketch_prism::ResolvedSketchPrism,
};
use oblikovati_session::{
    DocumentSession, TransactionObserver,
    part_evaluation::{PartCheckpointSlot, PartEvaluationSlot},
};

struct CheckpointObserver;
impl TransactionObserver for CheckpointObserver {
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

struct CountingEvaluator<'a> {
    kernel: CertifiedNamedSolidEvaluator<'a>,
    prisms: usize,
    blends: usize,
}
impl<'a> CountingEvaluator<'a> {
    fn new(history: &'a DocumentEditHistory) -> Self {
        Self {
            kernel: CertifiedNamedSolidEvaluator::new(history, document(), certified_resolution()),
            prisms: 0,
            blends: 0,
        }
    }
}
impl PartFeatureEvaluator for CountingEvaluator<'_> {
    type Body = NamedSolid;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, body: &NamedSolid, key: &[u8]) -> Result<(), String> {
        self.kernel.verify_body_reference(body, key)
    }
    fn resolve_planar_face(&self, body: &NamedSolid, key: &[u8]) -> Result<SketchPlane, String> {
        self.kernel.resolve_planar_face(body, key)
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedSolid, EvaluationFailure> {
        self.prisms += 1;
        self.kernel.evaluate_prism(feature, input)
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.blends += 1;
        self.kernel
            .evaluate_blend(kind, feature, body, edges, distance)
    }
}
fn definition(edge: Vec<u8>, radius: f64) -> FilletDefinition {
    FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(radius)).unwrap()
}
fn replace_fillet(
    snapshot: &mut DocumentSnapshot,
    definition: FilletDefinition,
) -> Result<(), String> {
    let reads = definition.reads();
    snapshot
        .work_geometry
        .replace(
            FeatureId(2),
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        )
        .map(|_| ())
        .map_err(|error| error.to_string())
}
#[test]
fn changed_named_fillet_replays_from_the_prism_checkpoint() {
    let mut history = owner(cube_document());
    let edge = certified(&history)
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
    let token = history.begin("Fillet").unwrap();
    assert_eq!(
        history
            .create_fillet(&token, "Fillet", definition(edge.clone(), 0.1))
            .unwrap(),
        FeatureId(2)
    );
    history.commit(&token, false).unwrap();
    let mut original = CountingEvaluator::new(&history);
    let checkpoints = history
        .snapshot()
        .evaluate_part_with_checkpoints(&mut original)
        .unwrap();
    assert_eq!((original.prisms, original.blends), (1, 1));
    let mut candidate = history.snapshot().clone();
    replace_fillet(&mut candidate, definition(edge.clone(), 0.15)).unwrap();
    let mut resumed_backend = CountingEvaluator::new(&history);
    let replay = checkpoints
        .replay_candidate_tail(&candidate, &mut resumed_backend)
        .unwrap();
    let mut full_backend = CountingEvaluator::new(&history);
    let full = candidate.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!((resumed_backend.prisms, resumed_backend.blends), (0, 1));
    assert_eq!((full_backend.prisms, full_backend.blends), (1, 1));
    assert_eq!(replay.health(), full.health());
    let actual = replay.body(FeatureId(2)).unwrap();
    let expected = full.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual.reference_keys(&history, document()).unwrap(),
        expected.reference_keys(&history, document()).unwrap()
    );
    for face in actual.face_reference_keys(&history, document()).unwrap() {
        assert!(
            (actual.face_area(&history, document(), &face).unwrap()
                - expected.face_area(&history, document(), &face).unwrap())
            .abs()
                < 1e-10
        );
    }
    drop(replay);
    let mut first_backend = CountingEvaluator::new(&history);
    let first = checkpoints
        .replay_with_checkpoints(&candidate, &mut first_backend, || false)
        .unwrap();
    assert_eq!((first_backend.prisms, first_backend.blends), (0, 1));
    let mut second_candidate = candidate.clone();
    replace_fillet(&mut second_candidate, definition(edge, 0.2)).unwrap();
    let mut second_backend = CountingEvaluator::new(&history);
    let second = first
        .replay_with_checkpoints(&second_candidate, &mut second_backend, || false)
        .unwrap();
    assert_eq!((second_backend.prisms, second_backend.blends), (0, 1));
    let mut baseline = CountingEvaluator::new(&history);
    let expected = second_candidate
        .evaluate_part_features(&mut baseline)
        .unwrap();
    let actual = second.result().body(FeatureId(2)).unwrap();
    let expected = expected.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual.reference_keys(&history, document()).unwrap(),
        expected.reference_keys(&history, document()).unwrap()
    );
    for face in actual.face_reference_keys(&history, document()).unwrap() {
        assert!(
            (actual.face_area(&history, document(), &face).unwrap()
                - expected.face_area(&history, document(), &face).unwrap())
            .abs()
                < 1e-10
        );
    }
    drop(original);
    drop(resumed_backend);
    drop(full_backend);
    drop(first_backend);
    drop(second_backend);
    drop(baseline);
    history
        .refresh(|snapshot| snapshot.parameters.set_expression(1, "20 mm"))
        .unwrap();
    assert!(
        checkpoints
            .result()
            .body(FeatureId(2))
            .unwrap()
            .volume(&history, document())
            .is_err()
    );
}

#[test]
fn session_accepts_checkpointed_named_fillet_and_invalidates_it_on_undo() {
    let mut session = DocumentSession::new(7, owner(cube_document()), CheckpointObserver);
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
    session
        .create_fillet("Fillet", definition(edge.clone(), 0.1))
        .unwrap();
    let mut checkpoints = PartCheckpointSlot::default();
    let mut evaluator = CountingEvaluator::new(session.history());
    let completed = session
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut evaluator)
        .unwrap();
    drop(evaluator);
    checkpoints.adopt(&session, completed).unwrap();
    let (captured, transition) = {
        let prefix = checkpoints.candidate_prefix(&session).unwrap();
        session
            .run_projected_with_transition(
                "Edit fillet",
                |snapshot, _| replace_fillet(snapshot, definition(edge, 0.15)),
                |snapshot, _, history| {
                    let mut backend = CountingEvaluator::new(history);
                    let candidate = prefix
                        .prepare_candidate(history, snapshot, &mut backend)?
                        .ok_or("expected changed fillet")?;
                    assert_eq!((backend.prisms, backend.blends), (0, 1));
                    candidate.capture(history, |_, body| {
                        body.capture_detached(history, document())
                    })
                },
            )
            .unwrap()
    };
    let completed = complete_named_candidate(captured, &transition, &session, document()).unwrap();
    let mut slot = PartEvaluationSlot::default();
    session.adopt_part_evaluation(completed, &mut slot).unwrap();
    let actual = slot.current(&session).unwrap().body(FeatureId(2)).unwrap();
    let mut backend = CountingEvaluator::new(session.history());
    let full = session
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let expected = full.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual
            .reference_keys(session.history(), document())
            .unwrap(),
        expected
            .reference_keys(session.history(), document())
            .unwrap()
    );
    for face in actual
        .face_reference_keys(session.history(), document())
        .unwrap()
    {
        assert!(
            (actual
                .face_area(session.history(), document(), &face)
                .unwrap()
                - expected
                    .face_area(session.history(), document(), &face)
                    .unwrap())
            .abs()
                < 1e-10
        );
    }
    drop(backend);
    session.undo().unwrap();
    assert!(slot.current(&session).is_err());
    assert!(checkpoints.candidate_prefix(&session).is_err());
}

fn commit_checkpointed_fillet(
    session: &mut DocumentSession<CheckpointObserver>,
    part: &mut PartEvaluationSlot<NamedSolid, EvaluationFailure>,
    checkpoints: &mut PartCheckpointSlot<NamedSolid, EvaluationFailure>,
    edge: &[u8],
    radius: f64,
) -> (usize, usize) {
    let foreign_history = owner(cube_document());
    let mut foreign_backend = CountingEvaluator::new(&foreign_history);
    let foreign_result = foreign_history
        .snapshot()
        .evaluate_part_features(&mut foreign_backend)
        .unwrap();
    let foreign_body = foreign_result.body(FeatureId(1)).unwrap();
    let ((captured, foreign_capture, counts), transition) = {
        let prefix = checkpoints.candidate_prefix(session).unwrap();
        session
            .run_projected_with_transition(
                "Edit fillet",
                |snapshot, _| replace_fillet(snapshot, definition(edge.to_vec(), radius)),
                |snapshot, _, history| {
                    let mut backend = CountingEvaluator::new(history);
                    let candidate = prefix
                        .prepare_checkpoints(history, snapshot, &mut backend)?
                        .ok_or("expected changed checkpoints")?;
                    if !candidate.has_current_source(FeatureId(2)) {
                        return Err("expected a live fillet descendant".into());
                    }
                    let captured = candidate.capture(history, |_, body| {
                        body.capture_detached(history, document())
                    })?;
                    let foreign_capture = candidate.capture(history, |_, _| {
                        foreign_body.capture_detached(&foreign_history, document())
                    })?;
                    Ok((captured, foreign_capture, (backend.prisms, backend.blends)))
                },
            )
            .unwrap()
    };
    assert!(
        complete_named_checkpoints(foreign_capture, &transition, session, document(),).is_err()
    );
    let completed = complete_named_checkpoints(captured, &transition, session, document()).unwrap();
    session
        .adopt_replayed_checkpoints(completed, part, checkpoints)
        .unwrap();
    counts
}

#[test]
fn accepted_named_checkpoints_survive_successive_fillet_edits() {
    let mut session = DocumentSession::new(7, owner(cube_document()), CheckpointObserver);
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
    session
        .create_fillet("Fillet", definition(edge.clone(), 0.1))
        .unwrap();
    let mut checkpoints = PartCheckpointSlot::default();
    let mut backend = CountingEvaluator::new(session.history());
    let completed = session
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut backend)
        .unwrap();
    drop(backend);
    checkpoints.adopt(&session, completed).unwrap();
    let mut part = PartEvaluationSlot::default();
    assert_eq!(
        commit_checkpointed_fillet(&mut session, &mut part, &mut checkpoints, &edge, 0.15),
        (0, 1)
    );
    assert_eq!(
        commit_checkpointed_fillet(&mut session, &mut part, &mut checkpoints, &edge, 0.2),
        (0, 1)
    );
    let actual = part.current(&session).unwrap().body(FeatureId(2)).unwrap();
    let mut backend = CountingEvaluator::new(session.history());
    let full = session
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let expected = full.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual
            .reference_keys(session.history(), document())
            .unwrap(),
        expected
            .reference_keys(session.history(), document())
            .unwrap()
    );
    for face in actual
        .face_reference_keys(session.history(), document())
        .unwrap()
    {
        assert!(
            (actual
                .face_area(session.history(), document(), &face)
                .unwrap()
                - expected
                    .face_area(session.history(), document(), &face)
                    .unwrap())
            .abs()
                < 1e-10
        );
    }
    drop(backend);
    session.undo().unwrap();
    assert!(part.current(&session).is_err());
    assert!(checkpoints.candidate_prefix(&session).is_err());
}
