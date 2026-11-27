// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus,
        part::{DeleteBodyDefinition, PartFeatureEvaluator, PartFeatureFailure},
        sketch_prism::ResolvedSketchPrism,
    },
    identity::SemanticReferenceName,
};
struct NamedReplayFake;
impl PartFeatureEvaluator for NamedReplayFake {
    type Body = Vec<u8>;
    type Failure = String;
    fn resolve_planar_face(
        &self,
        _body: &Self::Body,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        Err(oblikovati_model::feature::part::unavailable_planar_face(
            key,
        ))
    }

    fn evaluate_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        _feature: oblikovati_model::feature::FeatureId,
        _body: &Self::Body,
        _edges: &[Vec<u8>],
        _distance: f64,
    ) -> Result<Self::Body, oblikovati_model::feature::part::PartFeatureFailure<Self::Failure>>
    {
        Err(kind.unsupported())
    }

    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        _: ResolvedSketchPrism<'_>,
    ) -> Result<Vec<u8>, String> {
        Ok(body_key(feature.0, 4))
    }
    fn verify_body_reference(&self, body: &Vec<u8>, key: &[u8]) -> Result<(), String> {
        if body.as_slice() == key {
            return Ok(());
        }
        Err(format!("fake target {key:?}: expected {body:?}"))
    }
}
fn body_key(feature: u64, kind: u32) -> Vec<u8> {
    SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        feature,
        kind,
        "sketch-prism",
        "body",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap()
}
fn delete_entry(snapshot: &mut DocumentSnapshot, feature: u64, key: Vec<u8>) {
    let definition = PartFeatureDefinition::DeleteBody(DeleteBodyDefinition::new(key).unwrap());
    let reads = definition.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(feature),
            "delete-body",
            definition,
            reads,
        ))
        .unwrap();
}
fn deletion_document() -> DocumentSnapshot {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    delete_entry(&mut snapshot, 3, body_key(2, 4));
    snapshot
}
#[test]
fn deletion_consumes_current_body_and_reports_its_exact_key() {
    let snapshot = deletion_document();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert!(result.body(FeatureId(2)).is_none());
    assert_eq!(result.ordered_bodies().count(), 0);
    assert_eq!(
        result.deleted_body(FeatureId(3)),
        Some(body_key(2, 4).as_slice())
    );
    assert!(
        result
            .health()
            .iter()
            .all(|r| r.status == FeatureEvaluationStatus::Healthy)
    );
}

#[test]
fn candidate_tail_replays_deletion_without_mutating_the_retained_prefix() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let source = owner.snapshot().clone();
    let retained = source.evaluate_part_features(&mut NamedReplayFake).unwrap();
    let mut candidate = source.clone();
    delete_entry(&mut candidate, 3, body_key(2, 4));
    let preview = retained
        .replay_candidate_tail(&candidate, &mut NamedReplayFake)
        .unwrap();
    let full = candidate
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert_eq!(preview.health(), full.health());
    assert_eq!(preview.body_count(), full.ordered_bodies().count());
    assert!(!preview.has_body(FeatureId(2)));
    assert!(retained.body(FeatureId(2)).is_some());
}

#[test]
fn candidate_tail_refuses_changed_prefix_inputs_and_retains_failures() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let source = owner.snapshot().clone();
    let retained = source.evaluate_part_features(&mut NamedReplayFake).unwrap();
    let mut candidate = source.clone();
    delete_entry(&mut candidate, 3, body_key(2, 4));
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert!(
        retained
            .replay_candidate_tail(&candidate, &mut NamedReplayFake)
            .is_err()
    );
    let mut invalid = source.clone();
    let wrong = SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        2,
        4,
        "other",
        "body",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap();
    delete_entry(&mut invalid, 3, wrong);
    let preview = retained
        .replay_candidate_tail(&invalid, &mut NamedReplayFake)
        .unwrap();
    let full = invalid
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert_eq!(preview.health(), full.health());
    assert_eq!(preview.body_count(), full.ordered_bodies().count());
    assert!(retained.body(FeatureId(2)).is_some());
}
#[test]
fn a_second_deletion_fails_locally_and_independent_feature_runs() {
    let mut snapshot = deletion_document();
    delete_entry(&mut snapshot, 4, body_key(2, 4));
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(5), "independent", prism_source(), &snapshot)
            .unwrap();
    snapshot.work_geometry.append(feature).unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert!(
        matches!(&result.health()[3].status, FeatureEvaluationStatus::Failed(PartFeatureFailure::BodyReference(cause)) if cause.contains("unconsumed"))
    );
    assert!(result.deleted_body(FeatureId(4)).is_none());
    assert_eq!(result.ordered_bodies().count(), 1);
}
#[test]
fn suppression_and_end_of_part_restore_body_without_rebinding() {
    let mut snapshot = deletion_document();
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(3), true)
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert_eq!(result.body(FeatureId(2)), Some(&body_key(2, 4)));
    assert!(result.deleted_body(FeatureId(3)).is_none());
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(3), false)
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(2)).unwrap();
    assert_eq!(
        snapshot
            .evaluate_part_features(&mut NamedReplayFake)
            .unwrap()
            .ordered_bodies()
            .count(),
        1
    );
}
#[test]
fn deletion_verifies_the_whole_name_and_preserves_body_on_failure() {
    let mut snapshot = deletion_document();
    let mut wrong_key = body_key(2, 4);
    *wrong_key.last_mut().unwrap() = 1;
    let definition =
        PartFeatureDefinition::DeleteBody(DeleteBodyDefinition::new(wrong_key).unwrap());
    snapshot
        .work_geometry
        .replace(FeatureId(3), definition.clone(), definition.reads())
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 1);
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::BodyReference(_))
    ));
    assert!(result.deleted_body(FeatureId(3)).is_none());
}
#[test]
fn deletion_recipe_is_versioned_and_requires_body_keys_and_source_reads() {
    assert!(DeleteBodyDefinition::new(vec![]).is_err());
    assert!(
        DeleteBodyDefinition::new(body_key(2, 1))
            .unwrap_err()
            .contains("kind 4")
    );
    let snapshot = deletion_document();
    let mut recipe = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 17);
    assert_eq!(recipe.features[2].definition.version, 12);
    let restored = FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert!(restored.same_work_plane_recipe(&snapshot.work_geometry));
    recipe.version = 16;
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
    recipe.version = 17;
    recipe.features[2].definition_reads = Default::default();
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
}
#[test]
fn suppressed_source_makes_deletion_fail_and_cannot_be_removed_while_referenced() {
    let mut snapshot = deletion_document();
    assert!(snapshot.work_geometry.remove(FeatureId(2)).is_err());
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::BodyReference(_))
    ));
    assert!(result.deleted_body(FeatureId(3)).is_none());
}

#[test]
fn failed_source_quarantines_deletion() {
    let mut snapshot = deletion_document();
    snapshot.parameters.set_expression(3, "0 cm").unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Blocked { .. }
    ));
    assert!(result.deleted_body(FeatureId(3)).is_none());
}
#[test]
fn body_deletion_is_not_a_datum_or_construction_feature() {
    let mut snapshot = deletion_document();
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    assert!(matches!(
        &datums.health[2].status,
        FeatureEvaluationStatus::Blocked { drivers } if drivers == &[FeatureId(2)]
    ));
    snapshot
        .work_geometry
        .set_work_plane_construction(FeatureId(3), true)
        .unwrap();
    assert!(snapshot.work_geometry.part_recipe().is_err());
    let result = snapshot
        .evaluate_part_features(&mut NamedReplayFake)
        .unwrap();
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Metadata(_))
    ));
    assert_eq!(result.ordered_bodies().count(), 1);
}
