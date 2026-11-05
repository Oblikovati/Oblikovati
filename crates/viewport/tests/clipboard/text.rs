// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn incremental_unicode_is_decoded_only_after_the_terminal_property() {
    let mut receive = IncrementalText::begin(TextEncoding::Utf8, 4, 16).unwrap();
    for chunk in [vec![0xf0], vec![0x9f, 0x99], vec![0x82]] {
        assert_eq!(
            receive.accept(TextEncoding::Utf8, &chunk).unwrap(),
            ReceiveStep::Pending
        );
    }
    assert_eq!(
        receive.accept(TextEncoding::Utf8, &[]).unwrap(),
        ReceiveStep::Complete("🙂".into())
    );
    assert!(receive.accept(TextEncoding::Utf8, &[]).is_err());
}
#[test]
fn advertised_length_is_a_lower_bound_not_an_exact_length() {
    let mut receive = IncrementalText::begin(TextEncoding::Utf8, 1, 8).unwrap();
    receive.accept(TextEncoding::Utf8, b"hello").unwrap();
    assert_eq!(
        receive.accept(TextEncoding::Utf8, &[]).unwrap(),
        ReceiveStep::Complete("hello".into())
    );
    let mut short = IncrementalText::begin(TextEncoding::Utf8, 3, 8).unwrap();
    short.accept(TextEncoding::Utf8, b"a").unwrap();
    assert!(short.accept(TextEncoding::Utf8, &[]).is_err());
}
#[test]
fn byte_budget_and_encoding_are_checked_before_appending() {
    assert!(IncrementalText::begin(TextEncoding::Utf8, 9, 8).is_err());
    let mut receive = IncrementalText::begin(TextEncoding::Utf8, 1, 3).unwrap();
    let mut wrong_encoding = IncrementalText::begin(TextEncoding::Utf8, 1, 3).unwrap();
    assert!(wrong_encoding.accept(TextEncoding::Latin1, b"a").is_err());
    assert!(wrong_encoding.accept(TextEncoding::Utf8, b"a").is_err());
    receive.accept(TextEncoding::Utf8, b"abc").unwrap();
    assert!(receive.accept(TextEncoding::Utf8, b"d").is_err());
    assert_eq!(receive.received, b"abc");
    assert!(receive.accept(TextEncoding::Utf8, &[]).is_err());
}
#[test]
fn gui_text_never_contains_nul_or_partial_utf8() {
    assert!(
        decode_text(TextEncoding::Utf8, b"a\0b", 8)
            .unwrap_err()
            .contains("byte 1")
    );
    assert!(decode_text(TextEncoding::Utf8, &[0xf0, 0x9f], 8).is_err());
    assert_eq!(decode_text(TextEncoding::Latin1, &[0xe9], 8).unwrap(), "é");
    assert_eq!(decode_text(TextEncoding::Utf8, b"", 8).unwrap(), "");
}

#[test]
fn latin1_expansion_is_bounded_before_utf8_storage_is_built() {
    assert!(decode_text(TextEncoding::Latin1, &[0xe9, 0xe9], 2).is_err());
    assert_eq!(
        decode_text(TextEncoding::Latin1, &[0xe9, 0xe9], 4).unwrap(),
        "éé"
    );
}
