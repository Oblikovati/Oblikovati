// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::VariantValue;
use oblikovati_model::attribute::{AttributeTarget, DocumentAttributes};
use oblikovati_persistence::metadata::{decode_attributes, encode_attributes};

// Go AttributeManager.Encode, fixed baseline aa8297d1, Go 1.27.0/Linux.
const FIXTURE: &[u8] = include_bytes!("fixtures/go-binary-attributes.bin");

#[test]
fn go_attributes_round_trip_with_empty_namespaces_and_empty_anchors() {
    let decoded = decode_attributes(FIXTURE).unwrap();
    assert_eq!(decoded.anchors().len(), 2);
    let first = &decoded.anchors()[0];
    assert_eq!(first.sets()[0].name, "Empty");
    assert!(first.sets()[0].attributes().is_empty());
    assert_eq!(
        decoded.get(&first.target, "AddIn", "tag").unwrap().value,
        VariantValue::String("µm".into())
    );
    assert_eq!(
        decoded.get(&first.target, "AddIn", "count").unwrap().value,
        VariantValue::Integer(-7)
    );
    assert!(decoded.anchors()[1].sets().is_empty());
    assert_eq!(encode_attributes(&decoded).unwrap(), FIXTURE);
    assert!(
        decoded.same_recipe(&decode_attributes(&encode_attributes(&decoded).unwrap()).unwrap())
    );
}

#[test]
fn malformed_counts_truncation_trailing_input_and_keys_are_rejected() {
    for end in 0..FIXTURE.len() {
        assert!(decode_attributes(&FIXTURE[..end]).is_err(), "length {end}");
    }
    let mut encoded = FIXTURE.to_vec();
    encoded.push(0);
    assert!(
        decode_attributes(&encoded)
            .unwrap_err()
            .to_string()
            .contains("trailing")
    );
    assert!(decode_attributes(&u32::MAX.to_le_bytes()).is_err());
    let mut bad_key = FIXTURE.to_vec();
    bad_key[12] = 99;
    assert!(decode_attributes(&bad_key).is_err());
    assert_eq!(
        decode_attributes(&[0; 4]).unwrap(),
        DocumentAttributes::default()
    );
}

#[test]
fn duplicate_identity_is_refused_after_legacy_envelope_normalization() {
    let legacy = [0u8; 16];
    let target = AttributeTarget::from_encoded(&legacy).unwrap();
    let mut encoded = 2u32.to_le_bytes().to_vec();
    for key in [legacy.as_slice(), target.encoded()] {
        encoded.extend_from_slice(&(key.len() as u32).to_le_bytes());
        encoded.extend_from_slice(key);
        encoded.extend_from_slice(&0u32.to_le_bytes());
    }
    assert!(
        decode_attributes(&encoded)
            .unwrap_err()
            .to_string()
            .contains("unique reference")
    );
    encoded[..4].copy_from_slice(&1u32.to_le_bytes());
    encoded.truncate(28);
    let restored = decode_attributes(&encoded).unwrap();
    assert_eq!(restored.anchors()[0].target, target);
    assert_eq!(&encode_attributes(&restored).unwrap()[8..13], b"OKRK\x01");
}

#[test]
fn duplicate_set_and_attribute_names_are_refused() {
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    let mut attributes = DocumentAttributes::default();
    assert!(attributes.ensure_target(&target));
    assert!(!attributes.ensure_target(&target));
    attributes.ensure_set(&target, "aaaa").unwrap();
    attributes.ensure_set(&target, "bbbb").unwrap();
    let mut encoded = encode_attributes(&attributes).unwrap();
    let position = encoded
        .windows(4)
        .position(|bytes| bytes == b"bbbb")
        .unwrap();
    encoded[position..position + 4].copy_from_slice(b"aaaa");
    assert!(decode_attributes(&encoded).is_err());
    for name in ["first", "other"] {
        attributes
            .put(&target, "aaaa", name, VariantValue::Boolean(true))
            .unwrap();
    }
    let mut encoded = encode_attributes(&attributes).unwrap();
    let position = encoded
        .windows(5)
        .position(|bytes| bytes == b"other")
        .unwrap();
    encoded[position..position + 5].copy_from_slice(b"first");
    assert!(decode_attributes(&encoded).is_err());
}
