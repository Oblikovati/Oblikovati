// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{json::Bytes, types::VariantValue};
use oblikovati_persistence::metadata::{decode_value, encode_value};

// Captured through Go PropertySets.EncodeProperties at aa8297d1b0ac33c024f19e54821536d76e8a447a.
fn fixtures() -> Vec<(VariantValue, &'static str)> {
    vec![
        (VariantValue::Boolean(false), "0000"),
        (VariantValue::Boolean(true), "0001"),
        (VariantValue::Integer(i64::MIN), "010000000000000080"),
        (VariantValue::Integer(i64::MAX), "01ffffffffffffff7f"),
        (VariantValue::Double(-0.0), "020000000000000080"),
        (VariantValue::Double(f64::MAX), "02ffffffffffffef7f"),
        (
            VariantValue::Double(f64::from_bits(1)),
            "020100000000000000",
        ),
        (VariantValue::String("µm\0".into()), "0304000000c2b56d00"),
        (
            VariantValue::Bytes(Bytes(Some(vec![0, 255, 42]))),
            "040300000000ff2a",
        ),
        (VariantValue::Bytes(Bytes(None)), "0400000000"),
        (VariantValue::String(String::new()), "0300000000"),
    ]
}

#[test]
fn every_value_matches_the_go_fixture_and_preserves_the_unread_suffix() {
    for (expected, hex) in fixtures() {
        let encoded = encode_value(&expected).unwrap();
        let actual_hex: String = encoded.iter().map(|byte| format!("{byte:02x}")).collect();
        assert_eq!(actual_hex, hex);
        let mut stream = encoded.clone();
        stream.extend_from_slice(&[254, 253]);
        let (decoded, rest) = decode_value(&stream).unwrap();
        assert_eq!(rest, [254, 253]);
        assert_eq!(encode_value(&decoded).unwrap(), encoded);
        assert_eq!(decoded, expected);
    }
}

#[test]
fn every_truncated_fixture_and_unbounded_length_is_rejected() {
    for (value, _) in fixtures() {
        let encoded = encode_value(&value).unwrap();
        for boundary in 0..encoded.len() {
            assert!(
                decode_value(&encoded[..boundary]).is_err(),
                "{value:?} at {boundary}"
            );
        }
    }
    for tag in [3, 4] {
        assert!(decode_value(&[tag, 255, 255, 255, 255]).is_err());
    }
}

#[test]
fn invalid_tags_booleans_utf8_and_nonfinite_values_are_refused() {
    for encoded in [vec![255], vec![0, 2], vec![3, 1, 0, 0, 0, 255]] {
        assert!(
            decode_value(&encoded)
                .unwrap_err()
                .to_string()
                .contains("expected")
        );
    }
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        assert!(encode_value(&VariantValue::Double(value)).is_err());
        let mut encoded = vec![2];
        encoded.extend_from_slice(&value.to_le_bytes());
        assert!(decode_value(&encoded).is_err());
    }
    assert!(encode_value(&VariantValue::Uninitialized).is_err());
}

#[test]
fn empty_bytes_normalize_to_go_nil_and_decoded_values_own_their_storage() {
    let empty = encode_value(&VariantValue::Bytes(Bytes(Some(Vec::new())))).unwrap();
    assert_eq!(
        decode_value(&empty).unwrap().0,
        VariantValue::Bytes(Bytes(None))
    );
    let mut stream = vec![4, 2, 0, 0, 0, 7, 8];
    let decoded = decode_value(&stream).unwrap().0;
    stream[5] = 99;
    assert_eq!(decoded, VariantValue::Bytes(Bytes(Some(vec![7, 8]))));
}
