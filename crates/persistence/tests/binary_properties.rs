// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{json::Bytes, types::VariantValue};
use oblikovati_model::property::{
    DocumentProperties, DocumentProperty, SUMMARY_INFORMATION, USER_DEFINED,
};
use oblikovati_persistence::metadata::{decode_properties, encode_properties};

// Generated with Go 1.27.0 from the fixed Go baseline aa8297d1, using EncodeProperties.
const GO_PROPERTIES: &[u8] = include_bytes!("fixtures/go-binary-properties.bin");

#[test]
fn binary_collections_match_go_order_values_empty_sets_and_provenance() {
    let mut expected = DocumentProperties::default();
    expected
        .put(
            SUMMARY_INFORMATION,
            "Title",
            VariantValue::String("µm\0".into()),
        )
        .unwrap();
    expected.ensure_set("Empty").unwrap();
    for (name, value) in [
        ("approved", VariantValue::Boolean(true)),
        ("count", VariantValue::Integer(i64::MIN)),
        ("zero", VariantValue::Double(-0.0)),
        ("bytes", VariantValue::Bytes(Bytes(Some(vec![0, 255])))),
        ("empty", VariantValue::Bytes(Bytes(None))),
    ] {
        expected.put("Fixture", name, value).unwrap();
    }
    expected
        .restore_property(
            USER_DEFINED,
            DocumentProperty {
                name: "width".into(),
                value: VariantValue::Double(2.5),
                from_parameter: "width".into(),
            },
        )
        .unwrap();
    assert_eq!(encode_properties(&expected).unwrap(), GO_PROPERTIES);
    let restored = decode_properties(GO_PROPERTIES).unwrap();
    assert!(restored.same_recipe(&expected));
    assert_eq!(encode_properties(&restored).unwrap(), GO_PROPERTIES);
}

#[test]
fn complete_collection_validation_rejects_truncation_trailing_and_bad_counts() {
    for end in 0..GO_PROPERTIES.len() {
        assert!(
            decode_properties(&GO_PROPERTIES[..end]).is_err(),
            "length {end}"
        );
    }
    let mut trailing = GO_PROPERTIES.to_vec();
    trailing.push(0);
    assert!(
        decode_properties(&trailing)
            .unwrap_err()
            .to_string()
            .contains("trailing")
    );
    assert!(decode_properties(&u32::MAX.to_le_bytes()).is_err());
    assert!(decode_properties(&0u32.to_le_bytes()).is_err());
}

#[test]
fn repeated_set_and_property_names_are_refused_instead_of_overwritten() {
    let mut properties = DocumentProperties::default();
    properties.ensure_set("aaaa").unwrap();
    properties.ensure_set("bbbb").unwrap();
    let mut encoded = encode_properties(&properties).unwrap();
    let offset = encoded
        .windows(4)
        .position(|value| value == b"bbbb")
        .unwrap();
    encoded[offset..offset + 4].copy_from_slice(b"aaaa");
    assert!(
        decode_properties(&encoded)
            .unwrap_err()
            .to_string()
            .contains("unique")
    );
    properties
        .put("aaaa", "first", VariantValue::Boolean(true))
        .unwrap();
    properties
        .put("aaaa", "other", VariantValue::Boolean(false))
        .unwrap();
    let mut encoded = encode_properties(&properties).unwrap();
    let offset = encoded
        .windows(5)
        .position(|value| value == b"other")
        .unwrap();
    encoded[offset..offset + 5].copy_from_slice(b"first");
    assert!(
        decode_properties(&encoded)
            .unwrap_err()
            .to_string()
            .contains("unique")
    );
}

#[test]
fn reordered_standard_sets_are_rejected_without_silent_normalization() {
    use oblikovati_model::property::{DESIGN_TRACKING, DOCUMENT_SUMMARY};
    let mut encoded = 4u32.to_le_bytes().to_vec();
    for name in [
        DOCUMENT_SUMMARY,
        SUMMARY_INFORMATION,
        DESIGN_TRACKING,
        USER_DEFINED,
    ] {
        encoded.extend_from_slice(&(name.len() as u32).to_le_bytes());
        encoded.extend_from_slice(name.as_bytes());
        encoded.extend_from_slice(&0u32.to_le_bytes());
    }
    assert!(
        decode_properties(&encoded)
            .unwrap_err()
            .to_string()
            .contains("standard sets")
    );
    let empty = DocumentProperties::default();
    assert_eq!(
        decode_properties(&encode_properties(&empty).unwrap()).unwrap(),
        empty
    );
}
