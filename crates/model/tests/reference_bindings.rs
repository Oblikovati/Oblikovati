// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
#[path = "reference/support.rs"]
mod support;
use oblikovati_model::{
    history::DocumentEditHistory,
    identity::{
        DocumentGuid, ExactReferenceBindings, RecipeIdAllocator, ReferenceEnvelope, ReferenceScheme,
    },
};
fn history() -> DocumentEditHistory {
    let snapshot = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}
fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn key(context: u64, kind: u32, payload: &[u8]) -> Vec<u8> {
    ReferenceEnvelope {
        scheme: ReferenceScheme::Current,
        context,
        kind,
        payload,
    }
    .encode_current()
    .unwrap()
}
fn bindings(owner: &DocumentEditHistory) -> ExactReferenceBindings<&'static str> {
    ExactReferenceBindings::from_entries(
        document(),
        owner.evaluation_snapshot().scope().clone(),
        [
            (key(2, 1, b"cap"), "face"),
            (key(2, 2, b"cap"), "edge"),
            (key(3, 1, b"cap"), "other context"),
        ],
    )
    .unwrap()
}
#[test]
fn context_kind_and_payload_all_participate_in_exact_binding() {
    let owner = history();
    let index = bindings(&owner);
    assert_eq!(
        *index
            .bind(&owner, document(), &key(2, 1, b"cap"), 1)
            .unwrap(),
        "face"
    );
    assert_eq!(
        *index
            .bind(&owner, document(), &key(2, 2, b"cap"), 2)
            .unwrap(),
        "edge"
    );
    assert_eq!(
        *index
            .bind(&owner, document(), &key(3, 1, b"cap"), 1)
            .unwrap(),
        "other context"
    );
    assert!(
        index
            .bind(&owner, document(), &key(2, 1, b"cap"), 2)
            .unwrap_err()
            .contains("expected 2")
    );
    assert!(
        index
            .bind(&owner, document(), &key(2, 1, b"cap/child"), 1)
            .is_err()
    );
    assert!(
        index
            .bind(&owner, document(), &key(4, 1, b"cap"), 1)
            .is_err()
    );
}
#[test]
fn legacy_and_current_names_match_and_duplicates_are_ambiguous() {
    let owner = history();
    let index = bindings(&owner);
    let current = key(2, 1, b"cap");
    assert_eq!(
        *index.bind(&owner, document(), &current[5..], 1).unwrap(),
        "face"
    );
    for names in [
        vec![current.clone(), current.clone()],
        vec![current.clone(), current[5..].to_vec()],
    ] {
        let refusal = ExactReferenceBindings::from_entries(
            document(),
            owner.evaluation_snapshot().scope().clone(),
            names
                .into_iter()
                .map(|name| (name, "same or different entity")),
        )
        .err()
        .unwrap();
        assert!(refusal.contains("one entity"));
    }
}
#[test]
fn enumeration_is_canonical_and_independent_of_input_order() {
    let owner = history();
    let keys = [key(8, 4, b"body"), key(2, 1, b"cap")];
    let first = ExactReferenceBindings::from_entries(
        document(),
        owner.evaluation_snapshot().scope().clone(),
        keys.iter().cloned().map(|name| (name, ())),
    )
    .unwrap();
    let second = ExactReferenceBindings::from_entries(
        document(),
        owner.evaluation_snapshot().scope().clone(),
        keys.iter().rev().map(|name| (name[5..].to_vec(), ())),
    )
    .unwrap();
    let names = first
        .encoded_keys(&owner, document())
        .unwrap()
        .collect::<Vec<_>>();
    assert_eq!(
        names,
        second
            .encoded_keys(&owner, document())
            .unwrap()
            .collect::<Vec<_>>()
    );
    for name in names {
        let kind = ReferenceEnvelope::decode(name).unwrap().kind;
        assert_eq!(first.bind(&owner, document(), name, kind), Ok(&()));
    }
}
#[test]
fn history_and_document_are_both_required_even_at_equal_generation() {
    let owner = history();
    let index = bindings(&owner);
    let other = history();
    let other_document = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    assert!(
        index
            .bind(&other, document(), &key(2, 1, b"cap"), 1)
            .is_err()
    );
    assert!(
        index
            .bind(&owner, other_document, &key(2, 1, b"cap"), 1)
            .is_err()
    );
    assert!(index.encoded_keys(&other, document()).is_err());
    assert!(index.encoded_keys(&owner, other_document).is_err());
}
#[test]
fn edits_and_owner_closure_refuse_stale_binding_and_admission() {
    let mut owner = history();
    let index = bindings(&owner);
    let old_scope = owner.evaluation_snapshot().scope().clone();
    owner
        .refresh(|snapshot| snapshot.parameters.set_expression(3, "3 cm"))
        .unwrap();
    assert!(
        index
            .bind(&owner, document(), &key(2, 1, b"cap"), 1)
            .is_err()
    );
    assert!(index.encoded_keys(&owner, document()).is_err());
    assert!(ExactReferenceBindings::<()>::from_entries(document(), old_scope, []).is_err());
    let closed_scope = owner.evaluation_snapshot().scope().clone();
    drop(owner);
    assert!(ExactReferenceBindings::<()>::from_entries(document(), closed_scope, []).is_err());
}
#[test]
fn malformed_and_unknown_kind_keys_never_enter_or_bind() {
    let owner = history();
    let index = bindings(&owner);
    let mut unknown_scheme = key(2, 1, b"cap");
    unknown_scheme[4] = 2;
    for invalid in [
        vec![],
        key(2, 0, b"cap"),
        key(2, 99, b"cap"),
        unknown_scheme,
    ] {
        assert!(index.bind(&owner, document(), &invalid, 1).is_err());
        assert!(
            ExactReferenceBindings::from_entries(
                document(),
                owner.evaluation_snapshot().scope().clone(),
                [(invalid, ())]
            )
            .is_err()
        );
    }
}

#[test]
fn go_fixture_identity_binds_without_interpreting_opaque_payload_bytes() {
    let owner = history();
    for row in include_str!("fixtures/go-reference-envelope.tsv")
        .lines()
        .filter(|row| !row.starts_with('#'))
    {
        let (legacy, current) = row.split_once('\t').unwrap();
        let legacy = support::hex_bytes(legacy);
        let current = support::hex_bytes(current);
        let envelope = ReferenceEnvelope::decode(&current).unwrap();
        if matches!(envelope.kind, 0 | u32::MAX) {
            continue;
        }
        let index = ExactReferenceBindings::from_entries(
            document(),
            owner.evaluation_snapshot().scope().clone(),
            [(legacy.clone(), "Go entity")],
        )
        .unwrap();
        assert_eq!(
            index
                .encoded_keys(&owner, document())
                .unwrap()
                .collect::<Vec<_>>(),
            vec![current.as_slice()]
        );
        assert_eq!(
            *index
                .bind(&owner, document(), &legacy, envelope.kind)
                .unwrap(),
            "Go entity"
        );
        assert_eq!(
            *index
                .bind(&owner, document(), &current, envelope.kind)
                .unwrap(),
            "Go entity"
        );
    }
}
#[test]
fn undo_does_not_revalidate_old_native_bindings() {
    let mut owner = history();
    let index = bindings(&owner);
    let token = owner.begin("Depth").unwrap();
    owner
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
    owner.undo().unwrap();
    assert!(
        index
            .bind(&owner, document(), &key(2, 1, b"cap"), 1)
            .is_err()
    );
    let rebuilt = bindings(&owner);
    assert_eq!(
        *rebuilt
            .bind(&owner, document(), &key(2, 1, b"cap"), 1)
            .unwrap(),
        "face"
    );
}
#[test]
fn explicit_legacy_migration_handles_magic_prefix_contexts_without_a_fallback() {
    let owner = history();
    let context = u64::from_le_bytes(*b"OKRKabcd");
    let current = key(context, 1, b"cap");
    let legacy = &current[5..];
    assert!(ReferenceEnvelope::decode(legacy).is_err());
    let upgraded = ReferenceEnvelope::decode_legacy(legacy)
        .unwrap()
        .encode_current()
        .unwrap();
    let index = ExactReferenceBindings::from_entries(
        document(),
        owner.evaluation_snapshot().scope().clone(),
        [(upgraded.clone(), "face")],
    )
    .unwrap();
    assert_eq!(
        *index.bind(&owner, document(), &upgraded, 1).unwrap(),
        "face"
    );
}

struct MutatingReferenceEntries<'a> {
    owner: Option<&'a mut DocumentEditHistory>,
}
impl Iterator for MutatingReferenceEntries<'_> {
    type Item = (Vec<u8>, ());
    fn next(&mut self) -> Option<Self::Item> {
        let owner = self.owner.take()?;
        owner
            .refresh(|snapshot| snapshot.parameters.set_expression(3, "3 cm"))
            .unwrap();
        Some((key(2, 1, b"cap"), ()))
    }
}
#[test]
fn admission_checks_generation_again_after_consuming_the_producer() {
    let mut owner = history();
    let scope = owner.evaluation_snapshot().scope().clone();
    let entries = MutatingReferenceEntries {
        owner: Some(&mut owner),
    };
    let refusal = ExactReferenceBindings::from_entries(document(), scope, entries)
        .err()
        .unwrap();
    assert!(refusal.contains("unchanged owner"));
}
