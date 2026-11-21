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

#[test]
fn checkpoint_restores_the_body_consumed_by_an_edited_fillet() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    let mut initial = CountingBackend::default();
    let checkpoints = source.evaluate_part_with_checkpoints(&mut initial).unwrap();
    assert_eq!((initial.prisms, initial.blends), (1, 1));
    assert_eq!(checkpoints.len(), 4);
    assert!(checkpoints.result().body(FeatureId(2)).is_none());
    let mut edited = source.clone();
    edited
        .work_geometry
        .set_suppressed(FeatureId(3), true)
        .unwrap();
    let mut backend = CountingBackend::default();
    let replay = checkpoints
        .replay_candidate_tail(&edited, &mut backend)
        .unwrap();
    let mut full_backend = CountingBackend::default();
    let full = edited.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 0));
    assert_eq!((full_backend.prisms, full_backend.blends), (1, 0));
    assert_eq!(replay.health(), full.health());
    assert_eq!(
        replay.body(FeatureId(2)).unwrap().keys,
        full.body(FeatureId(2)).unwrap().keys
    );
    assert!(replay.body(FeatureId(3)).is_none());
}

#[test]
fn checkpoint_replays_changed_and_dependent_blends_in_order() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    add_fillet_edge(&mut source, 4, 3, "other-edge");
    let checkpoints = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut edited = source.clone();
    let definition = FilletDefinition::new(
        key(2, 4, "body"),
        vec![key(2, 2, "edge")],
        DatumLength::Constant(0.5),
    )
    .unwrap();
    let reads = definition.reads();
    edited
        .work_geometry
        .replace(
            FeatureId(3),
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        )
        .unwrap();
    let mut backend = CountingBackend::default();
    let replay = checkpoints
        .replay_candidate_tail(&edited, &mut backend)
        .unwrap();
    let mut full_backend = CountingBackend::default();
    let full = edited.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 2));
    assert_eq!((full_backend.prisms, full_backend.blends), (1, 2));
    assert_eq!(replay.health(), full.health());
    let actual = replay.body(FeatureId(4)).unwrap();
    let expected = full.body(FeatureId(4)).unwrap();
    assert_eq!(
        replay.body_from_source(FeatureId(3)).unwrap().keys,
        expected.keys
    );
    assert_eq!(actual.keys, expected.keys);
    assert_eq!(actual.deleted, expected.deleted);
    assert_eq!(actual.radius, expected.radius);
}

#[test]
fn checkpoint_replay_preserves_failed_and_blocked_diagnostics() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    add_fillet_edge(&mut source, 4, 3, "other-edge");
    let mut backend = CountingBackend::default();
    backend.inner.reject = true;
    let checkpoints = source.evaluate_part_with_checkpoints(&mut backend).unwrap();
    let mut full_backend = CountingBackend::default();
    full_backend.inner.reject = true;
    let full = source.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!(checkpoints.result().health(), full.health());
    let mut edited = source.clone();
    edited
        .work_geometry
        .set_suppressed(FeatureId(4), true)
        .unwrap();
    let replay = checkpoints
        .replay_candidate_tail(&edited, &mut CountingBackend::default())
        .unwrap();
    let mut full_backend = CountingBackend::default();
    full_backend.inner.reject = true;
    let expected = edited.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!(replay.health(), expected.health());
    assert!(replay.body(FeatureId(2)).is_some());
}

#[test]
fn checkpoint_replay_discards_partial_state_on_cancellation() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    let mut polls = 0;
    let result =
        source.evaluate_part_with_checkpoints_cancellable(&mut CountingBackend::default(), || {
            polls += 1;
            polls > 2
        });
    assert!(matches!(
        result,
        Err(oblikovati_model::feature::part::PartFeatureProgramError::Cancelled { .. })
    ));
}

#[test]
fn checkpoint_handles_end_of_part_truncation_without_running_the_kernel() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    let checkpoints = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut edited = source.clone();
    edited.work_geometry.set_end_of_part(Some(2)).unwrap();
    let mut backend = CountingBackend::default();
    let replay = checkpoints
        .replay_candidate_tail(&edited, &mut backend)
        .unwrap();
    let full = edited
        .evaluate_part_features(&mut CountingBackend::default())
        .unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 0));
    assert_eq!(replay.health(), full.health());
    assert_eq!(
        replay.body(FeatureId(2)).unwrap().keys,
        full.body(FeatureId(2)).unwrap().keys
    );
    assert!(
        checkpoints
            .replay_candidate_tail(&source, &mut backend)
            .is_err()
    );
}

fn replace_radius(
    snapshot: &mut DocumentSnapshot,
    feature: u64,
    source: u64,
    edge: &str,
    radius: f64,
) {
    let definition = FilletDefinition::new(
        key(source, 4, "body"),
        vec![key(2, 2, edge)],
        DatumLength::Constant(radius),
    )
    .unwrap();
    let reads = definition.reads();
    snapshot
        .work_geometry
        .replace(
            FeatureId(feature),
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        )
        .unwrap();
}

#[test]
fn replayed_checkpoints_support_successive_edits_without_rebuilding_the_prism() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    add_fillet_edge(&mut source, 4, 3, "other-edge");
    let original = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut first = source.clone();
    replace_radius(&mut first, 3, 2, "edge", 0.5);
    let mut backend = CountingBackend::default();
    let first_checkpoints = original
        .replay_with_checkpoints(&first, &mut backend, || false)
        .unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 2));
    let mut second = first.clone();
    replace_radius(&mut second, 4, 3, "other-edge", 0.25);
    let mut backend = CountingBackend::default();
    let second_checkpoints = first_checkpoints
        .replay_with_checkpoints(&second, &mut backend, || false)
        .unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 1));
    let full = second
        .evaluate_part_features(&mut CountingBackend::default())
        .unwrap();
    assert_eq!(second_checkpoints.result().health(), full.health());
    let actual = second_checkpoints.result().body(FeatureId(4)).unwrap();
    let expected = full.body(FeatureId(4)).unwrap();
    assert_eq!(actual.keys, expected.keys);
    assert_eq!(actual.deleted, expected.deleted);
    assert_eq!(actual.radius, expected.radius);
    assert!(original.result().body(FeatureId(4)).is_some());
}

#[test]
fn cancelled_checkpoint_suffix_does_not_replace_the_source() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    let checkpoints = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut candidate = source.clone();
    replace_radius(&mut candidate, 3, 2, "edge", 0.5);
    let mut backend = CountingBackend::default();
    assert!(
        checkpoints
            .replay_with_checkpoints(&candidate, &mut backend, || true)
            .is_err()
    );
    assert_eq!((backend.prisms, backend.blends), (0, 0));
    assert!(checkpoints.result().body(FeatureId(3)).is_some());
}

#[test]
fn replayed_checkpoints_truncate_at_end_of_part_and_can_resume_again() {
    let mut source = document();
    add_fillet(&mut source, 3, 2);
    let original = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut truncated = source.clone();
    truncated.work_geometry.set_end_of_part(Some(2)).unwrap();
    let mut backend = CountingBackend::default();
    let cutoff = original
        .replay_with_checkpoints(&truncated, &mut backend, || false)
        .unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 0));
    assert_eq!(cutoff.len(), 3);
    assert!(cutoff.result().body(FeatureId(2)).is_some());
    truncated.work_geometry.set_end_of_part(None).unwrap();
    let mut backend = CountingBackend::default();
    let resumed = cutoff
        .replay_with_checkpoints(&truncated, &mut backend, || false)
        .unwrap();
    assert_eq!((backend.prisms, backend.blends), (0, 1));
    assert!(resumed.result().body(FeatureId(3)).is_some());
    assert_eq!(original.result().health(), resumed.result().health());
}

#[test]
fn parameter_driven_fillet_replays_only_its_dirty_suffix() {
    let mut source = document();
    source
        .parameters
        .insert_expression(4, "radius", oblikovati_api::types::USER_PARAM, "1 mm")
        .unwrap();
    let definition = FilletDefinition::new(
        key(2, 4, "body"),
        vec![key(2, 2, "edge")],
        DatumLength::Parameter(4),
    )
    .unwrap();
    let reads = definition.reads();
    source
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    let checkpoints = source
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let mut edited = source.clone();
    edited.parameters.set_expression(4, "2 mm").unwrap();
    let mut resumed_backend = CountingBackend::default();
    let replay = checkpoints
        .replay_with_checkpoints(&edited, &mut resumed_backend, || false)
        .unwrap();
    let mut full_backend = CountingBackend::default();
    let full = edited.evaluate_part_features(&mut full_backend).unwrap();
    assert_eq!((resumed_backend.prisms, resumed_backend.blends), (0, 1));
    assert_eq!((full_backend.prisms, full_backend.blends), (1, 1));
    assert_eq!(replay.result().health(), full.health());
    let actual = replay.result().body(FeatureId(3)).unwrap();
    let expected = full.body(FeatureId(3)).unwrap();
    assert_eq!(actual.keys, expected.keys);
    assert_eq!(actual.deleted, expected.deleted);
    assert_eq!(actual.radius, expected.radius);
}

#[test]
fn prepared_checkpoint_bodies_publish_with_their_exact_candidate() {
    use oblikovati_model::{
        history::{DocumentEditHistory, PreparedValue, publish_prepared},
        identity::RecipeIdAllocator,
    };
    let snapshot = document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let (candidate, ()) = history
        .prepare_private_edit("fillet", |snapshot, ids, _| {
            assert_eq!(ids.allocate_feature().unwrap(), FeatureId(3));
            add_fillet(snapshot, 3, 2);
            Ok::<_, String>(())
        })
        .unwrap();
    let replay = candidate
        .snapshot()
        .evaluate_part_with_checkpoints(&mut CountingBackend::default())
        .unwrap();
    let prepared = replay
        .try_map_bodies(|_, body| Ok::<_, String>(PreparedValue::new(&candidate, body.clone())))
        .unwrap();
    let (published, _) = publish_prepared(candidate, prepared).unwrap();
    assert_eq!(published.len(), replay.len());
    assert_eq!(
        published.result().ordered_bodies().count(),
        replay.result().ordered_bodies().count()
    );
    assert_eq!(history.undo_labels(), ["fillet"]);
}
