// SPDX-License-Identifier: GPL-2.0-only
use super::super::requestor::finish_requestor;
use super::*;
fn property(kind: Atom, format: u8, bytes: &[u8]) -> GetPropertyReply {
    GetPropertyReply {
        format,
        sequence: 0,
        length: 0,
        type_: kind,
        bytes_after: 0,
        value_len: if format == 32 {
            bytes.len() as u32 / 4
        } else {
            bytes.len() as u32
        },
        value: bytes.to_vec(),
    }
}
#[test]
fn target_classifier_prefers_advertised_utf8_and_rejects_wrong_property_shapes() {
    let utf8 = 100;
    let bytes: Vec<u8> = [u32::from(AtomEnum::STRING), utf8]
        .iter()
        .flat_map(|atom| atom.to_ne_bytes())
        .collect();
    assert_eq!(
        classify_text_target(&property(AtomEnum::ATOM.into(), 32, &bytes), utf8).unwrap(),
        TextEncoding::Utf8
    );
    assert!(classify_text_target(&property(AtomEnum::STRING.into(), 8, b"utf8"), utf8).is_err());
    assert!(classify_text_target(&property(AtomEnum::ATOM.into(), 32, &[]), utf8).is_err());
}
#[test]
fn direct_property_decoder_checks_target_and_format_before_text() {
    assert_eq!(
        decode_text_property(TextEncoding::Utf8, 100, &property(100, 8, b"ok"), 2).unwrap(),
        "ok"
    );
    assert!(decode_text_property(TextEncoding::Utf8, 100, &property(101, 8, b"ok"), 2).is_err());
    assert!(decode_text_property(TextEncoding::Utf8, 100, &property(100, 32, &[0; 4]), 4).is_err());
}
#[test]
fn incremental_header_requires_one_word_and_cleanup_preserves_both_failures() {
    assert_eq!(
        incremental_lower_bound(&property(100, 32, &4_u32.to_ne_bytes())).unwrap(),
        4
    );
    assert!(incremental_lower_bound(&property(100, 8, &[4])).is_err());
    assert!(incremental_lower_bound(&property(100, 32, &[0; 8])).is_err());
    assert_eq!(
        finish_requestor::<String>(Ok("owned".into()), Ok(())).unwrap(),
        "owned"
    );
    let error =
        finish_requestor::<String>(Err("transfer fault".into()), Err("cleanup fault".into()))
            .unwrap_err();
    assert!(error.contains("transfer fault") && error.contains("cleanup fault"));
}
