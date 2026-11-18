// SPDX-License-Identifier: GPL-2.0-only
//! Exact named bodies survive a clean checkpoint within one history generation.
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid};
use oblikovati_model::feature::{
    fillet::FilletDefinition,
    part::{PartBlendKind, PartFeatureEvaluator, PartFeatureFailure},
    sketch_prism::ResolvedSketchPrism,
};

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
    replace_fillet(&mut candidate, definition(edge, 0.15)).unwrap();
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
    drop(original);
    drop(resumed_backend);
    drop(full_backend);
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
