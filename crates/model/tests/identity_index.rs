// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{
    DocumentGuid, DocumentIdentityIndex, FileIdentity, GuidSource, IdentityRegistration,
    IdentityRegistrationError, IdentityTransitionError,
};

struct FixedGuidSource {
    outcome: Result<DocumentGuid, IdentityTransitionError>,
    calls: usize,
}

impl GuidSource for FixedGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.calls += 1;
        self.outcome.clone()
    }
}

fn guid(number: u64) -> DocumentGuid {
    format!("00000000-0000-4000-8000-{number:012x}")
        .parse()
        .unwrap()
}

fn identity(number: u64) -> FileIdentity {
    FileIdentity {
        internal_name: guid(number),
        revision_id: Some(guid(100)),
        database_revision_id: Some(guid(101)),
        save_counter: 7,
        version_created: "alpha".into(),
        version_saved: "beta".into(),
        model_digest: None,
    }
}

fn source(number: u64) -> FixedGuidSource {
    FixedGuidSource {
        outcome: Ok(guid(number)),
        calls: 0,
    }
}

#[test]
fn unique_registration_does_not_consume_randomness() {
    let mut index = DocumentIdentityIndex::default();
    let mut random = source(9);
    assert_eq!(
        index.register(1, identity(1), &mut random).unwrap(),
        IdentityRegistration::Preserved
    );
    assert_eq!(index.identity(1), Some(&identity(1)));
    assert_eq!(index.owner(guid(1)), Some(1));
    assert_eq!(random.calls, 0);
    assert_eq!(index.identity(9), None);
    assert_eq!(index.owner(guid(9)), None);
}

#[test]
fn copied_file_collision_returns_explicit_reassignment_and_preserves_revisions() {
    let mut index = DocumentIdentityIndex::default();
    let mut random = source(2);
    let _first = index.register(1, identity(1), &mut random).unwrap();
    let outcome = index.register(2, identity(1), &mut random).unwrap();
    assert_eq!(
        outcome,
        IdentityRegistration::Reassigned {
            previous: guid(1),
            current: guid(2)
        }
    );
    let mut expected = identity(1);
    expected.internal_name = guid(2);
    assert_eq!(index.identity(2), Some(&expected));
    assert_eq!(index.identity(1), Some(&identity(1)));
    assert_eq!(index.owner(guid(1)), Some(1));
    assert_eq!(index.owner(guid(2)), Some(2));
    assert_eq!(random.calls, 1);
}

#[test]
fn close_releases_the_original_identity_for_reopen() {
    let mut index = DocumentIdentityIndex::default();
    let mut random = source(2);
    let _first = index.register(1, identity(1), &mut random).unwrap();
    assert_eq!(index.close(1).unwrap(), identity(1));
    assert_eq!(index.owner(guid(1)), None);
    assert_eq!(index.identity(1), None);
    let outcome = index.register(3, identity(1), &mut random).unwrap();
    assert_eq!(outcome, IdentityRegistration::Preserved);
    assert_eq!(index.owner(guid(1)), Some(3));
    assert_eq!(
        index.close(99),
        Err(IdentityRegistrationError::MissingDocument(99))
    );
}

#[test]
fn failed_collision_generation_leaves_both_indexes_unchanged() {
    let mut index = DocumentIdentityIndex::default();
    let mut random = source(2);
    let _first = index.register(1, identity(1), &mut random).unwrap();
    let _other = index.register(3, identity(3), &mut random).unwrap();
    for repeated in [1, 3, 100, 101] {
        random.outcome = Ok(guid(repeated));
        assert!(index.register(2, identity(1), &mut random).is_err());
        assert_eq!(index.identity(2), None);
        assert_eq!(index.identity(1), Some(&identity(1)));
        assert_eq!(index.owner(guid(3)), Some(3));
    }
    random.outcome = Err(IdentityTransitionError::Randomness("unavailable".into()));
    let error = index.register(2, identity(1), &mut random).unwrap_err();
    assert!(error.to_string().contains("unavailable"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(index.identity(2), None);
}

#[test]
fn duplicate_or_zero_session_ids_are_refused_before_generation() {
    let mut index = DocumentIdentityIndex::default();
    let mut random = source(2);
    assert_eq!(
        index.register(0, identity(1), &mut random),
        Err(IdentityRegistrationError::InvalidSessionId(0))
    );
    let _first = index.register(1, identity(1), &mut random).unwrap();
    assert_eq!(
        index.register(1, identity(2), &mut random),
        Err(IdentityRegistrationError::AlreadyRegistered(1))
    );
    assert_eq!(random.calls, 0);
    assert_eq!(index.identity(1), Some(&identity(1)));
    assert_eq!(index.owner(guid(2)), None);
}

#[test]
fn session_errors_report_the_offending_id_and_expected_state() {
    let errors = [
        IdentityRegistrationError::InvalidSessionId(0),
        IdentityRegistrationError::AlreadyRegistered(7),
        IdentityRegistrationError::MissingDocument(9),
    ];
    for (error, id) in errors.into_iter().zip([0, 7, 9]) {
        assert!(error.to_string().contains(&id.to_string()));
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
    }
}
