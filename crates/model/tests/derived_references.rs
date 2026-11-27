// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::identity::{
    DerivedReferenceName, DocumentGuid, ReferenceEnvelope, ReferenceScheme, SemanticReferenceName,
    topology_reference_document,
};

fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn parent(feature: u64, document: DocumentGuid) -> Vec<u8> {
    SemanticReferenceName::new(document, feature, 1, "prism", "face", &[1])
        .unwrap()
        .encode()
        .unwrap()
}

#[test]
fn derived_name_is_order_independent_and_retains_full_parent_names() {
    let first = parent(1, document());
    let second = parent(2, document());
    let a = DerivedReferenceName::new(
        document(),
        3,
        2,
        "intersection",
        &[first.clone(), second.clone()],
    )
    .unwrap();
    let b = DerivedReferenceName::new(document(), 3, 2, "intersection", &[second, first]).unwrap();
    let encoded = a.encode().unwrap();
    assert_eq!(encoded, b.encode().unwrap());
    assert_eq!(DerivedReferenceName::decode(&encoded).unwrap(), a);
    assert_eq!(topology_reference_document(&encoded).unwrap(), document());
    assert_eq!(
        ReferenceEnvelope::decode(&encoded).unwrap().scheme,
        ReferenceScheme::Derived
    );
}

#[test]
fn derived_names_reject_foreign_duplicate_and_malformed_parents() {
    let own = parent(1, document());
    let foreign = parent(1, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap());
    for parents in [
        vec![],
        vec![own.clone(), own.clone()],
        vec![own.clone(), foreign],
        vec![b"invalid".to_vec()],
    ] {
        assert!(DerivedReferenceName::new(document(), 3, 2, "intersection", &parents).is_err());
    }
    assert!(
        DerivedReferenceName::new(document(), 0, 2, "intersection", std::slice::from_ref(&own))
            .is_err()
    );
    assert!(
        DerivedReferenceName::new(
            document(),
            3,
            99,
            "intersection",
            std::slice::from_ref(&own)
        )
        .is_err()
    );
    assert!(DerivedReferenceName::new(document(), 3, 2, "Bad Tag", &[own]).is_err());
}

#[test]
fn decoder_rejects_truncation_and_noncanonical_parent_order() {
    let first = parent(1, document());
    let second = parent(2, document());
    let encoded = DerivedReferenceName::new(document(), 3, 2, "intersection", &[first, second])
        .unwrap()
        .encode()
        .unwrap();
    let envelope = ReferenceEnvelope::decode(&encoded).unwrap();
    for end in 0..envelope.payload.len() {
        let truncated = ReferenceEnvelope {
            payload: &envelope.payload[..end],
            ..envelope
        }
        .encode_current()
        .unwrap();
        assert!(
            DerivedReferenceName::decode(&truncated).is_err(),
            "accepted {end} bytes"
        );
    }
    let mut trailing = envelope.payload.to_vec();
    trailing.push(0);
    let bad = ReferenceEnvelope {
        payload: &trailing,
        ..envelope
    }
    .encode_current()
    .unwrap();
    assert!(DerivedReferenceName::decode(&bad).is_err());
    let offset = 16 + 1 + "intersection".len() + 1;
    let first_length =
        u16::from_le_bytes(envelope.payload[offset..offset + 2].try_into().unwrap()) as usize + 2;
    let mut reversed = envelope.payload[..offset].to_vec();
    reversed.extend_from_slice(&envelope.payload[offset + first_length..]);
    reversed.extend_from_slice(&envelope.payload[offset..offset + first_length]);
    let bad = ReferenceEnvelope {
        payload: &reversed,
        ..envelope
    }
    .encode_current()
    .unwrap();
    assert!(DerivedReferenceName::decode(&bad).is_err());
}

#[test]
fn nested_names_are_bounded_and_can_be_used_by_feature_definitions() {
    use oblikovati_model::feature::{
        fillet::FilletDefinition, part::DeleteBodyDefinition, work_geometry::DatumLength,
        work_plane::WorkPlaneFaceReference,
    };
    let root = parent(1, document());
    let face = DerivedReferenceName::new(
        document(),
        2,
        1,
        "generated-face",
        std::slice::from_ref(&root),
    )
    .unwrap()
    .encode()
    .unwrap();
    let edge = DerivedReferenceName::new(document(), 2, 2, "edge", std::slice::from_ref(&face))
        .unwrap()
        .encode()
        .unwrap();
    let body = DerivedReferenceName::new(document(), 2, 4, "body", std::slice::from_ref(&root))
        .unwrap()
        .encode()
        .unwrap();
    assert!(WorkPlaneFaceReference::new(face).is_ok());
    assert!(DeleteBodyDefinition::new(body.clone()).is_ok());
    assert!(FilletDefinition::new(body, vec![edge], DatumLength::Constant(1.)).is_ok());
    let mut nested = root;
    for feature in 2..10 {
        nested = DerivedReferenceName::new(document(), feature, 1, "child", &[nested])
            .unwrap()
            .encode()
            .unwrap();
    }
    assert!(DerivedReferenceName::new(document(), 10, 1, "child", &[nested]).is_err());
}

#[test]
fn derived_names_reject_oversized_parent_sets() {
    let mut parents = Vec::new();
    for feature in 1..=16 {
        let make_parent = |id| {
            SemanticReferenceName::new(document(), id, 1, "prism", "face", &[1; 64])
                .unwrap()
                .encode()
                .unwrap()
        };
        let pair = [make_parent(feature * 2), make_parent(feature * 2 + 1)];
        parents.push(
            DerivedReferenceName::new(document(), feature + 100, 1, "pair", &pair)
                .unwrap()
                .encode()
                .unwrap(),
        );
    }
    assert!(DerivedReferenceName::new(document(), 500, 2, "edge", &parents).is_err());
    parents.push(parent(900, document()));
    assert!(DerivedReferenceName::new(document(), 500, 2, "edge", &parents).is_err());
}
