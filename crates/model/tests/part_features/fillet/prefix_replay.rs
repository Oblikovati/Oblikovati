// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[derive(Default)]
struct CountingBackend {
    inner: NamedFilletBackend,
    prisms: usize,
    blends: usize,
}
impl PartFeatureEvaluator for CountingBackend {
    type Body = NamedBody;
    type Failure = String;
    fn verify_body_reference(&self, body: &NamedBody, key: &[u8]) -> Result<(), String> {
        self.inner.verify_body_reference(body, key)
    }
    fn resolve_planar_face(&self, body: &NamedBody, key: &[u8]) -> Result<SketchPlane, String> {
        self.inner.resolve_planar_face(body, key)
    }
    fn evaluate_prism(
        &mut self,
        id: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedBody, String> {
        self.prisms += 1;
        self.inner.evaluate_prism(id, input)
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        id: FeatureId,
        body: &NamedBody,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedBody, PartFeatureFailure<String>> {
        self.blends += 1;
        self.inner.evaluate_blend(kind, id, body, edges, distance)
    }
}

#[test]
fn fillet_candidate_reuses_its_retained_prism_and_matches_full_replay() {
    let source = document();
    let retained = source
        .evaluate_part_features(&mut CountingBackend::default())
        .unwrap();
    let mut candidate = source.clone();
    add_fillet(&mut candidate, 3, 2);
    let mut resumed_backend = CountingBackend::default();
    let resumed = retained
        .replay_candidate_tail(&candidate, &mut resumed_backend)
        .unwrap();
    let mut full_backend = CountingBackend::default();
    let full = candidate.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!((resumed_backend.prisms, resumed_backend.blends), (0, 1));
    assert_eq!((full_backend.prisms, full_backend.blends), (1, 1));
    assert_eq!(resumed.health(), full.health());
    assert_eq!(resumed.body_count(), full.ordered_bodies().count());
    assert!(resumed.has_body(FeatureId(3)));
    assert!(retained.body(FeatureId(2)).is_some());
}
