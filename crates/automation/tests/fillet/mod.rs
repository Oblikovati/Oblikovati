// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::prism_test_fixture as fixture;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    feature::work_geometry::DatumLength,
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
};
use oblikovati_session::{DocumentSession, TransactionObserver};

struct SilentFilletObserver;
impl TransactionObserver for SilentFilletObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
struct AcceptFillet;
impl FilletValidation for AcceptFillet {
    fn validate(&self, candidate: &DocumentSnapshot, feature: FeatureId) -> Result<(), String> {
        let feature = candidate
            .work_geometry
            .get(feature)
            .map_err(|cause| cause.to_string())?;
        if feature.definition().kind() != "fillet" {
            return Err("candidate: expected a fillet".into());
        }
        Ok(())
    }
}
struct RejectFillet;
impl FilletValidation for RejectFillet {
    fn validate(&self, _: &DocumentSnapshot, _: FeatureId) -> Result<(), String> {
        Err("kernel refusal".into())
    }
}
fn session() -> DocumentSession<SilentFilletObserver> {
    let snapshot = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentSession::new(
        1,
        DocumentEditHistory::new(snapshot, ids).unwrap(),
        SilentFilletObserver,
    )
}
fn key(kind: u32) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 1, kind, "fixture", "source", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn prepare(snapshot: &DocumentSnapshot, radius: &str) -> Result<PreparedFillet, Error> {
    prepare_fillet(snapshot, key(4), vec![key(2)], radius)
}

#[test]
fn prepare_accepts_lengths_and_parameter_names_and_refuses_invalid_radius() {
    let owner = session();
    assert_eq!(
        prepare(owner.snapshot(), "10 mm")
            .unwrap()
            .definition
            .radius(),
        DatumLength::Constant(1.)
    );
    assert_eq!(
        prepare(owner.snapshot(), "width")
            .unwrap()
            .definition
            .radius(),
        DatumLength::Parameter(1)
    );
    for value in ["0 mm", "-2 mm", "1 deg", "width + 1 mm"] {
        assert!(
            prepare(owner.snapshot(), value).is_err(),
            "radius {value:?}"
        );
    }
    assert!(prepare_fillet(owner.snapshot(), key(4), vec![], "2 mm").is_err());
}

#[test]
fn preview_does_not_publish_and_validated_commit_is_undoable() {
    let mut owner = session();
    let before = owner.generation();
    let prepared = prepare(owner.snapshot(), "2 mm").unwrap();
    let preview = owner
        .preview_projected(
            |snapshot, ids| prepared.clone().create(snapshot, ids),
            |candidate, id| PreparedFillet::validate(&AcceptFillet, candidate, id),
        )
        .unwrap();
    assert_eq!(owner.generation(), before);
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
    let committed = owner
        .run_projected(
            "Fillet",
            |snapshot, ids| prepared.create(snapshot, ids),
            |candidate, id| PreparedFillet::validate(&AcceptFillet, candidate, id),
        )
        .unwrap();
    assert_eq!(preview, committed);
    owner.undo().unwrap();
    assert!(owner.snapshot().work_geometry.get(committed).is_err());
    owner.redo().unwrap();
    assert!(owner.snapshot().work_geometry.get(committed).is_ok());
}

#[test]
fn rejected_candidate_does_not_change_document_or_history() {
    let mut owner = session();
    let before = owner.generation();
    let prepared = prepare(owner.snapshot(), "2 mm").unwrap();
    let result = owner.run_projected(
        "Fillet",
        |snapshot, ids| prepared.create(snapshot, ids),
        |candidate, id| PreparedFillet::validate(&RejectFillet, candidate, id),
    );
    assert!(result.is_err());
    assert_eq!(owner.generation(), before);
    assert!(owner.undo_labels().is_empty());
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
}

#[test]
fn certified_validator_checks_real_geometry_before_publication() {
    use oblikovati_martensite::{PrismResolution, PrismResolutionPolicy};
    let mut owner = session();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let body = SemanticReferenceName::new(document, 1, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let edge = |curve| {
        SemanticReferenceName::new(
            document,
            1,
            2,
            "sketch-prism",
            "start-edge-curve",
            &[1, curve],
        )
        .unwrap()
        .encode()
        .unwrap()
    };
    let policy = PrismResolutionPolicy::Fixed(PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    });
    let validator = CertifiedFilletValidation::new(document, policy).unwrap();
    assert!(validator.validate(owner.snapshot(), FeatureId(1)).is_err());
    let invalid = prepare_fillet(owner.snapshot(), body.clone(), vec![edge(999)], "2 mm").unwrap();
    let result = owner.run_projected(
        "Fillet",
        |snapshot, ids| invalid.create(snapshot, ids),
        |candidate, id| PreparedFillet::validate(&validator, candidate, id),
    );
    assert!(result.is_err());
    assert!(owner.undo_labels().is_empty());
    let valid = prepare_fillet(owner.snapshot(), body, vec![edge(101)], "2 mm").unwrap();
    let preview = owner
        .preview_projected(
            |snapshot, ids| valid.clone().create(snapshot, ids),
            |candidate, id| PreparedFillet::validate(&validator, candidate, id),
        )
        .unwrap();
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
    let committed = owner
        .run_projected(
            "Fillet",
            |snapshot, ids| valid.create(snapshot, ids),
            |candidate, id| PreparedFillet::validate(&validator, candidate, id),
        )
        .unwrap();
    assert_eq!(preview, committed);
    owner.undo().unwrap();
    assert!(owner.snapshot().work_geometry.get(committed).is_err());
}
