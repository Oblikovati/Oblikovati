// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{ReferenceEnvelope, ReferenceEnvelopeError, ReferenceScheme};

#[path = "reference/support.rs"]
mod support;
use support::hex_bytes;

#[test]
fn legacy_upgrade_matches_go_bytes_exactly() {
    let fixture = include_str!("fixtures/go-reference-envelope.tsv");
    let rows: Vec<&str> = fixture
        .lines()
        .filter(|row| !row.starts_with('#'))
        .collect();
    assert_eq!(rows.len(), 15);
    for row in rows {
        let (legacy_hex, current_hex) = row.split_once('\t').unwrap();
        let legacy = hex_bytes(legacy_hex);
        let current = hex_bytes(current_hex);
        let before = ReferenceEnvelope::decode(&legacy).unwrap();
        let after = ReferenceEnvelope::decode(&current).unwrap();
        assert_eq!(before.scheme, ReferenceScheme::Legacy);
        assert_eq!(after.scheme, ReferenceScheme::Current);
        assert_eq!(before.encode_current().unwrap(), current);
        assert_eq!(after.encode_current().unwrap(), current);
        assert_eq!(
            (before.context, before.kind, before.payload),
            (after.context, after.kind, after.payload)
        );
    }
}

#[test]
fn every_truncation_and_extra_byte_is_refused() {
    let envelope = ReferenceEnvelope {
        scheme: ReferenceScheme::Current,
        context: 5,
        kind: 21,
        payload: b"opaque\0key",
    };
    let encoded = envelope.encode_current().unwrap();
    for length in 0..encoded.len() {
        assert!(ReferenceEnvelope::decode(&encoded[..length]).is_err());
    }
    let mut extended = encoded.clone();
    extended.push(0);
    assert!(matches!(
        ReferenceEnvelope::decode(&extended),
        Err(ReferenceEnvelopeError::PayloadLength { .. })
    ));
    let mut false_length = encoded.clone();
    false_length[17..21].copy_from_slice(&u32::MAX.to_le_bytes());
    assert!(matches!(
        ReferenceEnvelope::decode(&false_length),
        Err(ReferenceEnvelopeError::PayloadLength { .. })
    ));
}

#[test]
fn unknown_schemes_are_not_reinterpreted_as_legacy() {
    let envelope = ReferenceEnvelope {
        scheme: ReferenceScheme::Current,
        context: 0,
        kind: 0,
        payload: &[],
    };
    let mut encoded = envelope.encode_current().unwrap();
    for scheme in [0, 2, 127, 255] {
        encoded[4] = scheme;
        assert_eq!(
            ReferenceEnvelope::decode(&encoded),
            Err(ReferenceEnvelopeError::UnsupportedScheme(scheme))
        );
    }
}

#[test]
fn explicit_legacy_decode_preserves_a_magic_prefix_context() {
    let mut legacy = [0_u8; 16];
    legacy[..4].copy_from_slice(b"OKRK");
    legacy[4] = 1;
    assert!(ReferenceEnvelope::decode(&legacy).is_err());
    let decoded = ReferenceEnvelope::decode_legacy(&legacy).unwrap();
    assert_eq!(
        decoded.context,
        u64::from_le_bytes(legacy[..8].try_into().unwrap())
    );
    let upgraded = decoded.encode_current().unwrap();
    let current = ReferenceEnvelope::decode(&upgraded).unwrap();
    assert_eq!(current.context, decoded.context);
    assert!(current.payload.is_empty());
}

#[test]
fn errors_name_the_bad_value_and_expected_shape() {
    let errors = [
        ReferenceEnvelopeError::ShortHeader {
            actual: 3,
            expected: 16,
        },
        ReferenceEnvelopeError::UnsupportedScheme(2),
        ReferenceEnvelopeError::PayloadLength {
            declared: 9,
            actual: 1,
        },
        ReferenceEnvelopeError::PayloadTooLarge(usize::MAX),
    ];
    for error in errors {
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
    }
}
