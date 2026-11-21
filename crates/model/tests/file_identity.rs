// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{
    DocumentGuid, FileIdentity, GuidSource, IdentityTransitionError, RecipeDigest, SystemGuidSource,
};
use std::collections::VecDeque;

struct SequenceGuidSource {
    outcomes: VecDeque<Result<DocumentGuid, IdentityTransitionError>>,
}

impl SequenceGuidSource {
    fn fresh() -> Self {
        let outcomes = (1..20)
            .map(|id| {
                Ok(format!("00000000-0000-4000-8000-{id:012x}")
                    .parse()
                    .unwrap())
            })
            .collect();
        Self { outcomes }
    }
}

impl GuidSource for SequenceGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.outcomes.pop_front().expect("unexpected GUID request")
    }
}

fn unsaved() -> (FileIdentity, SequenceGuidSource) {
    let mut source = SequenceGuidSource::fresh();
    let identity = FileIdentity::new("alpha", &mut source).unwrap();
    (identity, source)
}

fn saved() -> (FileIdentity, SequenceGuidSource) {
    let (identity, mut source) = unsaved();
    let digest = Some(RecipeDigest::of_recipe(b"recipe"));
    let saved = identity.prepare_save(digest, "beta", &mut source).unwrap();
    (saved, source)
}

#[test]
fn creation_has_no_save_revision() {
    let (initial, _) = unsaved();
    assert_eq!(initial.save_counter, 0);
    assert_eq!(initial.revision_id, None);
    assert_eq!(initial.database_revision_id, None);
    assert_eq!(initial.model_digest, None);
    assert_eq!(initial.version_created, "alpha");
    assert_eq!(initial.version_saved, "");
}

#[test]
fn save_preserves_document_identity_and_does_not_adopt_itself() {
    let (initial, mut source) = unsaved();
    let saved = initial.prepare_save(None, "beta", &mut source).unwrap();
    assert_eq!(saved.internal_name, initial.internal_name);
    assert_eq!((saved.save_counter, initial.save_counter), (1, 0));
    assert!(saved.revision_id.is_some());
    assert!(saved.database_revision_id.is_some());
    assert_ne!(saved.revision_id, saved.database_revision_id);
    assert_eq!(saved.version_created, "alpha");
    assert_eq!(saved.version_saved, "beta");
}

#[test]
fn recipe_changes_control_model_revision_but_every_save_changes_file_revision() {
    let (first, mut source) = saved();
    let digest = first.model_digest;
    let second = first.prepare_save(digest, "beta", &mut source).unwrap();
    assert_ne!(first.revision_id, second.revision_id);
    assert_eq!(first.database_revision_id, second.database_revision_id);
    assert_eq!(second.save_counter, 2);
    let changed = Some(RecipeDigest::of_recipe(b"edit"));
    let third = second.prepare_save(changed, "beta", &mut source).unwrap();
    assert_ne!(second.database_revision_id, third.database_revision_id);
    assert_eq!(third.model_digest, changed);
}

#[test]
fn unknown_recipe_digest_never_claims_the_model_is_unchanged() {
    let (known, mut source) = saved();
    let unknown = known.prepare_save(None, "beta", &mut source).unwrap();
    let again = unknown.prepare_save(None, "beta", &mut source).unwrap();
    assert_ne!(known.database_revision_id, unknown.database_revision_id);
    assert_ne!(unknown.database_revision_id, again.database_revision_id);
    assert_eq!(again.model_digest, None);
    let recovered = again
        .prepare_save(known.model_digest, "beta", &mut source)
        .unwrap();
    assert_ne!(again.database_revision_id, recovered.database_revision_id);
}

#[test]
fn saved_copy_has_an_independent_identity_with_the_same_model() {
    let (original, mut source) = saved();
    let copy = original.prepare_copy("gamma", &mut source).unwrap();
    assert_ne!(copy.internal_name, original.internal_name);
    assert_ne!(copy.revision_id, original.revision_id);
    assert_eq!(copy.database_revision_id, original.database_revision_id);
    assert_eq!(copy.model_digest, original.model_digest);
    assert_eq!(copy.save_counter, 1);
    assert_eq!(copy.version_created, "gamma");
    assert_eq!(copy.version_saved, "gamma");
    assert_eq!(original.version_created, "alpha");
    assert_eq!(original.version_saved, "beta");
}

#[test]
fn unsaved_copy_gets_all_three_distinct_identity_stamps() {
    let mut source = SequenceGuidSource::fresh();
    let original = FileIdentity::new("alpha", &mut source).unwrap();
    let copy = original.prepare_copy("beta", &mut source).unwrap();
    assert_ne!(Some(copy.internal_name), copy.revision_id);
    assert_ne!(Some(copy.internal_name), copy.database_revision_id);
    assert_ne!(copy.revision_id, copy.database_revision_id);
    assert!(copy.database_revision_id.is_some());
    assert_eq!(copy.model_digest, None);
}

#[test]
fn entropy_failure_leaves_the_source_identity_unchanged() {
    let mut source = SequenceGuidSource::fresh();
    let original = FileIdentity::new("alpha", &mut source).unwrap();
    let snapshot = original.clone();
    for success_count in 0..3 {
        let mut failing = SequenceGuidSource::fresh();
        failing.outcomes.pop_front();
        failing.outcomes.truncate(success_count);
        failing
            .outcomes
            .push_back(Err(IdentityTransitionError::Randomness(
                "test unavailable".into(),
            )));
        assert!(original.prepare_copy("beta", &mut failing).is_err());
        assert_eq!(original, snapshot);
    }
}

#[test]
fn repeated_guids_are_refused() {
    let (original, mut source) = saved();
    for forbidden in [
        Some(original.internal_name),
        original.revision_id,
        original.database_revision_id,
    ] {
        let guid = forbidden.unwrap();
        source.outcomes.push_front(Ok(guid));
        let error = original
            .prepare_save(None, "beta", &mut source)
            .unwrap_err();
        assert_eq!(error, IdentityTransitionError::RepeatedGuid(guid));
        assert!(error.to_string().contains(&guid.to_string()));
    }
}

#[test]
fn counter_overflow_is_refused_before_requesting_any_guids() {
    let (mut exhausted, mut source) = unsaved();
    exhausted.save_counter = u64::MAX;
    let remaining = source.outcomes.len();
    let overflow = exhausted
        .prepare_save(None, "beta", &mut source)
        .unwrap_err();
    assert_eq!(overflow, IdentityTransitionError::CounterOverflow(u64::MAX));
    assert!(overflow.to_string().contains(&u64::MAX.to_string()));
    assert_eq!(source.outcomes.len(), remaining);
}

#[test]
fn recipe_digest_matches_sha256_known_vectors() {
    assert_eq!(
        RecipeDigest::of_recipe(b"").to_string(),
        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    );
    assert_eq!(
        RecipeDigest::of_recipe(b"abc").to_string(),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
    assert_ne!(
        RecipeDigest::of_recipe(b"a\nb"),
        RecipeDigest::of_recipe(b"a b")
    );
}

#[test]
fn operating_system_source_produces_canonical_v4_stamps() {
    let guid = SystemGuidSource.generate().unwrap().to_string();
    assert_eq!(guid.len(), 36);
    assert_eq!(&guid[14..15], "4");
    assert!(matches!(&guid[19..20], "8" | "9" | "a" | "b"));
    let failure = IdentityTransitionError::Randomness("test unavailable".into());
    assert!(failure.to_string().contains("test unavailable"));
    assert!(failure.to_string().contains("expected 16 random bytes"));
    assert!(std::error::Error::source(&failure).is_none());
}

#[test]
fn save_entropy_failure_does_not_change_the_original() {
    let (original, mut source) = unsaved();
    let snapshot = original.clone();
    source.outcomes.truncate(1);
    source
        .outcomes
        .push_back(Err(IdentityTransitionError::Randomness("missing".into())));
    assert!(original.prepare_save(None, "beta", &mut source).is_err());
    assert_eq!(original, snapshot);
}

#[test]
fn repeated_new_revision_is_refused_within_the_same_save() {
    let (original, mut source) = unsaved();
    let next = source.outcomes.front().unwrap().clone();
    source.outcomes.insert(1, next);
    assert!(matches!(
        original.prepare_save(None, "beta", &mut source),
        Err(IdentityTransitionError::RepeatedGuid(_))
    ));
}

#[test]
fn snapshot_copy_preserves_model_stamp_only_for_a_known_matching_digest() {
    let (original, mut source) = saved();
    let same = original
        .prepare_copy_for_recipe(original.model_digest, "copy", &mut source)
        .unwrap();
    assert_eq!(same.database_revision_id, original.database_revision_id);
    for digest in [Some(RecipeDigest::of_recipe(b"edited")), None] {
        let copy = original
            .prepare_copy_for_recipe(digest, "copy", &mut source)
            .unwrap();
        assert_ne!(copy.database_revision_id, original.database_revision_id);
        assert_eq!(copy.model_digest, digest);
        assert_ne!(copy.internal_name, original.internal_name);
        assert_eq!(copy.save_counter, 1);
    }
}
