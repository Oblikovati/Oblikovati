// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{
    DocumentGuid, DocumentIdentityIndex, DocumentSaveError, DocumentSaveTarget, FileIdentity,
    GuidSource, IdentityTransitionError, RecipeDigest,
};
use std::io;

struct CountingGuidSource {
    next: u64,
}

impl GuidSource for CountingGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.next += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.next)
            .parse()
            .unwrap())
    }
}

#[derive(Default)]
struct FakeAtomicSaveTarget {
    committed: Option<FileIdentity>,
    attempts: Vec<FileIdentity>,
    fail: bool,
}

impl DocumentSaveTarget for FakeAtomicSaveTarget {
    type Failure = io::Error;

    fn persist(&mut self, candidate: &FileIdentity) -> Result<(), Self::Failure> {
        self.attempts.push(candidate.clone());
        if self.fail {
            return Err(io::Error::other("fake atomic write refused"));
        }
        self.committed = Some(candidate.clone());
        Ok(())
    }
}

fn open_document(counter: u64) -> (DocumentIdentityIndex, CountingGuidSource) {
    let mut source = CountingGuidSource { next: 0 };
    let mut identity = FileIdentity::new("alpha", &mut source).unwrap();
    identity.save_counter = counter;
    let mut index = DocumentIdentityIndex::default();
    let _registration = index.register(1, identity, &mut source).unwrap();
    (index, source)
}

#[test]
fn successful_save_adopts_exactly_the_persisted_identity() {
    let (mut index, mut source) = open_document(0);
    let guid = index.identity(1).unwrap().internal_name;
    let mut target = FakeAtomicSaveTarget::default();
    let digest = Some(RecipeDigest::of_recipe(b"recipe"));
    index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap();
    assert_eq!(index.identity(1), target.committed.as_ref());
    assert_eq!(index.identity(1).unwrap().save_counter, 1);
    assert_eq!(index.identity(1).unwrap().model_digest, digest);
    assert_eq!(index.owner(guid), Some(1));
    assert_eq!(target.attempts.len(), 1);
}

#[test]
fn failed_write_preserves_identity_and_retry_uses_fresh_revisions() {
    let (mut index, mut source) = open_document(0);
    let before = index.identity(1).unwrap().clone();
    let mut target = FakeAtomicSaveTarget {
        fail: true,
        ..Default::default()
    };
    let error = index
        .save(1, None, "beta", &mut source, &mut target)
        .unwrap_err();
    assert!(matches!(error, DocumentSaveError::Storage(_)));
    assert!(error.to_string().contains("fake atomic write refused"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(index.identity(1), Some(&before));
    assert_eq!(index.owner(before.internal_name), Some(1));
    assert_eq!(target.committed, None);
    target.fail = false;
    index
        .save(1, None, "beta", &mut source, &mut target)
        .unwrap();
    assert_eq!(index.identity(1).unwrap().save_counter, 1);
    assert_ne!(
        target.attempts[0].revision_id,
        target.attempts[1].revision_id
    );
}

#[test]
fn unchanged_recipe_retains_model_revision_through_the_storage_boundary() {
    let (mut index, mut source) = open_document(0);
    let mut target = FakeAtomicSaveTarget::default();
    let digest = Some(RecipeDigest::of_recipe(b"recipe"));
    index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap();
    let first = target.committed.clone().unwrap();
    index
        .save(1, digest, "gamma", &mut source, &mut target)
        .unwrap();
    let second = target.committed.as_ref().unwrap();
    assert_eq!(first.database_revision_id, second.database_revision_id);
    assert_ne!(first.revision_id, second.revision_id);
    assert_eq!(second.save_counter, 2);
    assert_eq!(second.version_saved, "gamma");
}

#[test]
fn missing_document_fails_before_generation_or_storage() {
    let (mut index, mut source) = open_document(0);
    let mut target = FakeAtomicSaveTarget::default();
    let error = index
        .save(9, None, "beta", &mut source, &mut target)
        .unwrap_err();
    assert!(matches!(error, DocumentSaveError::MissingDocument(9)));
    assert!(error.to_string().contains("9"));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(source.next, 1);
    assert!(target.attempts.is_empty());
}

#[test]
fn preparation_failure_never_reaches_storage_or_changes_the_registered_identity() {
    let (mut index, mut source) = open_document(u64::MAX);
    let before = index.identity(1).unwrap().clone();
    let mut target = FakeAtomicSaveTarget::default();
    let error = index
        .save(1, None, "beta", &mut source, &mut target)
        .unwrap_err();
    assert!(matches!(
        error,
        DocumentSaveError::Preparation(IdentityTransitionError::CounterOverflow(_))
    ));
    assert!(error.to_string().contains(&u64::MAX.to_string()));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(index.identity(1), Some(&before));
    assert!(target.attempts.is_empty());
    assert_eq!(source.next, 1);
}

#[test]
fn changed_recipe_commits_a_new_model_revision() {
    let (mut index, mut source) = open_document(0);
    let mut target = FakeAtomicSaveTarget::default();
    for recipe in [b"before".as_slice(), b"after".as_slice()] {
        let digest = Some(RecipeDigest::of_recipe(recipe));
        index
            .save(1, digest, "beta", &mut source, &mut target)
            .unwrap();
    }
    assert_ne!(
        target.attempts[0].database_revision_id,
        target.attempts[1].database_revision_id
    );
    assert_eq!(index.identity(1), target.committed.as_ref());
}

#[test]
fn copy_rejects_missing_source_and_another_live_document_guid_before_storage() {
    let (mut index, mut source) = open_document(0);
    let other = FileIdentity::new("alpha", &mut source).unwrap();
    let other_guid = other.internal_name;
    let _registration = index.register(2, other, &mut source).unwrap();
    let mut target = FakeAtomicSaveTarget::default();
    assert!(matches!(
        index.save_copy(99, None, "beta", &mut source, &mut target),
        Err(DocumentSaveError::MissingDocument(99))
    ));
    source.next = 1;
    let error = index
        .save_copy(1, None, "beta", &mut source, &mut target)
        .unwrap_err();
    assert!(
        matches!(error, DocumentSaveError::Preparation(IdentityTransitionError::RepeatedGuid(found)) if found == other_guid)
    );
    assert!(target.attempts.is_empty());
    assert_eq!(index.owner(other_guid), Some(2));
}
