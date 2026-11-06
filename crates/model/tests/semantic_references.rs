// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::identity::{
    DocumentGuid, ReferenceEnvelope, ReferenceScheme, SemanticReferenceName,
};
fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn name() -> SemanticReferenceName {
    SemanticReferenceName::new(document(), 7, 1, "sketch-prism", "side", &[1, 101]).unwrap()
}
#[test]
fn semantic_encoding_has_fixed_uuid_tags_and_little_endian_sources() {
    let bytes = name().encode().unwrap();
    let envelope = ReferenceEnvelope::decode(&bytes).unwrap();
    assert_eq!(envelope.scheme, ReferenceScheme::Semantic);
    assert_eq!((envelope.context, envelope.kind), (7, 1));
    let mut expected = vec![
        0x11, 0x11, 0x11, 0x11, 0x22, 0x22, 0x33, 0x33, 0x44, 0x44, 0x55, 0x55, 0x55, 0x55, 0x55,
        0x55,
    ];
    expected.extend_from_slice(b"\x0c\x00sketch-prism\x04\x00side\x02\x00");
    expected.extend_from_slice(&1_u64.to_le_bytes());
    expected.extend_from_slice(&101_u64.to_le_bytes());
    assert_eq!(envelope.payload, expected);
    assert_eq!(envelope.encode_current().unwrap(), bytes);
    assert_eq!(SemanticReferenceName::decode(&bytes).unwrap(), name());
    assert_eq!(name().document(), document());
}
#[test]
fn names_refuse_invalid_tags_ids_and_source_limits() {
    for tag in ["", "Upper", "bad/name", "é"] {
        assert!(SemanticReferenceName::new(document(), 1, 1, tag, "side", &[]).is_err());
        assert!(SemanticReferenceName::new(document(), 1, 1, "prism", tag, &[]).is_err());
    }
    assert!(SemanticReferenceName::new(document(), 0, 1, "prism", "side", &[1]).is_err());
    assert!(SemanticReferenceName::new(document(), 1, 99, "prism", "side", &[1]).is_err());
    assert!(SemanticReferenceName::new(document(), 1, 1, "prism", "side", &[0]).is_err());
    assert!(SemanticReferenceName::new(document(), 1, 1, &"a".repeat(65), "side", &[1]).is_err());
    assert!(SemanticReferenceName::new(document(), 1, 1, "prism", "side", &[1; 65]).is_err());
    let boundary = SemanticReferenceName::new(
        document(),
        u64::MAX,
        4,
        &"a".repeat(64),
        &"b".repeat(64),
        &[u64::MAX; 64],
    )
    .unwrap();
    assert_eq!(
        SemanticReferenceName::decode(&boundary.encode().unwrap()).unwrap(),
        boundary
    );
}
#[test]
fn malformed_semantic_payload_and_legacy_meaning_are_refused() {
    let bytes = name().encode().unwrap();
    let original = ReferenceEnvelope::decode(&bytes).unwrap();
    for length in 0..original.payload.len() {
        let short = ReferenceEnvelope {
            payload: &original.payload[..length],
            ..original
        }
        .encode_current()
        .unwrap();
        assert!(SemanticReferenceName::decode(&short).is_err(), "{length}");
    }
    let mut trailing = original.payload.to_vec();
    trailing.push(0);
    let bad = ReferenceEnvelope {
        payload: &trailing,
        ..original
    }
    .encode_current()
    .unwrap();
    assert!(SemanticReferenceName::decode(&bad).is_err());
    let legacy = ReferenceEnvelope {
        scheme: ReferenceScheme::Current,
        ..original
    }
    .encode_current()
    .unwrap();
    assert!(SemanticReferenceName::decode(&legacy).is_err());
    let mut bad_tag = original.payload.to_vec();
    bad_tag[18] = 255;
    let bad = ReferenceEnvelope {
        payload: &bad_tag,
        ..original
    }
    .encode_current()
    .unwrap();
    assert!(SemanticReferenceName::decode(&bad).is_err());
}

#[test]
fn decoded_counts_and_source_values_obey_the_constructor_limits() {
    let bytes = name().encode().unwrap();
    let original = ReferenceEnvelope::decode(&bytes).unwrap();
    // Fixed offsets come from the independent scheme-2 byte fixture above.
    for (offset, replacement) in [
        (16, vec![0, 0]),
        (16, vec![65, 0]),
        (36, vec![65, 0]),
        (38, vec![0; 8]),
    ] {
        let mut payload = original.payload.to_vec();
        payload[offset..offset + replacement.len()].copy_from_slice(&replacement);
        let invalid = ReferenceEnvelope {
            payload: &payload,
            ..original
        }
        .encode_current()
        .unwrap();
        assert!(
            SemanticReferenceName::decode(&invalid).is_err(),
            "offset {offset}"
        );
    }
    assert!(SemanticReferenceName::decode(b"bad").is_err());
}
