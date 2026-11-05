// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::attachment::{AttachmentContents, DocumentAttachments};
use oblikovati_persistence::package::DocumentPackage;
use std::time::{Duration, SystemTime};

fn package(records: &str) -> DocumentPackage {
    DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nattachments: {records}\n").as_bytes(),
    )
    .unwrap()
}

#[test]
fn go_attachment_fixture_round_trips_owned_payload_time_visibility_and_order() {
    let original = DocumentPackage::decode(include_bytes!("fixtures/go-attachments.yml")).unwrap();
    let mut attachments = original.restore_attachments().unwrap();
    assert_eq!(
        attachments
            .entries()
            .iter()
            .map(|entry| entry.name())
            .collect::<Vec<_>>(),
        ["generic", "linked", "embedded"]
    );
    assert!(!attachments.get("generic").unwrap().browser_visible());
    let linked = attachments.get("linked").unwrap();
    assert!(matches!(
        linked.contents(),
        AttachmentContents::Linked {
            last_known: Some(_)
        }
    ));
    assert!(
        matches!(attachments.get("embedded").unwrap().contents(), AttachmentContents::Embedded { bytes, .. } if bytes == &[0, 255, 128])
    );
    attachments.set_browser_visible("embedded", false).unwrap();
    let captured = original.with_attachments(&attachments).unwrap();
    let reopened = DocumentPackage::decode(&captured.encode().unwrap()).unwrap();
    assert_eq!(reopened.restore_attachments().unwrap(), attachments);
    assert_eq!(
        reopened.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(reopened.streams(), original.streams());
    let cleared = reopened
        .with_attachments(&DocumentAttachments::default())
        .unwrap();
    assert!(cleared.section_yaml("attachments").unwrap().is_none());
}

#[test]
fn timestamps_preserve_nanoseconds_before_and_after_epoch_and_normalize_offsets() {
    for (text, expected) in [
        (
            "1970-01-01T00:00:00.000000001Z",
            SystemTime::UNIX_EPOCH + Duration::from_nanos(1),
        ),
        (
            "1969-12-31T23:59:59.999999999Z",
            SystemTime::UNIX_EPOCH - Duration::from_nanos(1),
        ),
        ("1970-01-01T01:30:00+01:30", SystemTime::UNIX_EPOCH),
        ("1970-01-01T00:00:00Z", SystemTime::UNIX_EPOCH),
    ] {
        let original = package(&format!(
            "[{{name: link, kind: linked, fullFileName: file, lastKnownFileTime: '{text}'}}]"
        ));
        let attachments = original.restore_attachments().unwrap();
        assert_eq!(
            attachments.get("link").unwrap().contents(),
            &AttachmentContents::Linked {
                last_known: Some(expected)
            }
        );
        assert_eq!(
            original
                .with_attachments(&attachments)
                .unwrap()
                .restore_attachments()
                .unwrap(),
            attachments
        );
    }
    let original = package("[{name: link, kind: linked, fullFileName: file}]");
    let attachments = original.restore_attachments().unwrap();
    assert_eq!(
        attachments.get("link").unwrap().contents(),
        &AttachmentContents::Linked { last_known: None }
    );
    assert_eq!(
        original
            .with_attachments(&attachments)
            .unwrap()
            .restore_attachments()
            .unwrap(),
        attachments
    );
}

#[test]
fn malformed_or_incompatible_attachment_records_cannot_be_overwritten() {
    let records = [
        "{name: '', kind: generic, fullFileName: file}",
        "{name: one, kind: generic}",
        "{name: one, kind: future, fullFileName: file}",
        "{name: one, kind: generic, fullFileName: file, extra: keep}",
        "{name: one, kind: linked, fullFileName: file, payload: AA==}",
        "{name: one, kind: generic, fullFileName: file, lastKnownFileTime: yesterday}",
        "{name: one, kind: embedded, fullFileName: file, resourceId: invalid}",
        "{name: one, kind: embedded, fullFileName: file, resourceId: 00000000-0000-4000-8000-000000000001, payload: '!'}",
        "{name: one, kind: generic, fullFileName: file}, {name: one, kind: generic, fullFileName: another}",
        "{name: one, kind: embedded, fullFileName: file, resourceId: 00000000-0000-4000-8000-000000000001}, {name: two, kind: embedded, fullFileName: another, resourceId: 00000000-0000-4000-8000-000000000001}",
    ];
    for record in records {
        let original = package(&format!("[{record}]"));
        assert!(original.restore_attachments().is_err(), "{record}");
        assert!(
            original
                .with_attachments(&DocumentAttachments::default())
                .is_err(),
            "{record}"
        );
    }
    for timestamp in [
        "yesterday",
        "2026-02-30T00:00:00Z",
        "2016-12-31T23:59:60Z",
        "2026-01-01T00:00:00.1234567891Z",
    ] {
        let original = package(&format!(
            "[{{name: link, kind: linked, fullFileName: file, lastKnownFileTime: '{timestamp}'}}]"
        ));
        let error = original.restore_attachments().unwrap_err();
        assert!(error.to_string().contains(timestamp));
    }
}

#[test]
fn empty_sections_and_empty_embedded_payloads_are_valid() {
    for section in ["null", "[]"] {
        assert!(
            package(section)
                .restore_attachments()
                .unwrap()
                .entries()
                .is_empty()
        );
    }
    let absent = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert!(absent.restore_attachments().unwrap().entries().is_empty());
    let empty = package(
        "[{name: empty, kind: embedded, fullFileName: file, resourceId: 00000000-0000-4000-8000-000000000001}]",
    );
    let attachments = empty.restore_attachments().unwrap();
    assert!(
        matches!(attachments.get("empty").unwrap().contents(), AttachmentContents::Embedded { bytes, .. } if bytes.is_empty())
    );
    assert_eq!(
        empty
            .with_attachments(&attachments)
            .unwrap()
            .restore_attachments()
            .unwrap(),
        attachments
    );
}

#[test]
fn unsupported_output_timestamp_refuses_capture_without_mutation() {
    let mut attachments = DocumentAttachments::default();
    let outside = SystemTime::UNIX_EPOCH
        .checked_add(Duration::from_secs(253_402_300_800))
        .unwrap();
    attachments
        .restore(
            "link",
            "file",
            AttachmentContents::Linked {
                last_known: Some(outside),
            },
            true,
        )
        .unwrap();
    let original = package("[]");
    assert!(
        original
            .with_attachments(&attachments)
            .unwrap_err()
            .to_string()
            .contains("timestamp")
    );
    assert!(original.restore_attachments().unwrap().entries().is_empty());
}

#[test]
fn go_zero_time_is_unknown_and_cannot_silently_replace_an_explicit_model_time() {
    for text in ["0001-01-01T00:00:00Z", "0001-01-01T01:00:00+01:00"] {
        let original = package(&format!(
            "[{{name: link, kind: linked, fullFileName: file, lastKnownFileTime: '{text}'}}]"
        ));
        assert_eq!(
            original
                .restore_attachments()
                .unwrap()
                .get("link")
                .unwrap()
                .contents(),
            &AttachmentContents::Linked { last_known: None }
        );
    }
    let zero = SystemTime::UNIX_EPOCH
        .checked_sub(Duration::from_secs(62_135_596_800))
        .unwrap();
    let mut attachments = DocumentAttachments::default();
    attachments
        .restore(
            "link",
            "file",
            AttachmentContents::Linked {
                last_known: Some(zero),
            },
            false,
        )
        .unwrap();
    assert!(
        package("[]")
            .with_attachments(&attachments)
            .unwrap_err()
            .to_string()
            .contains("zero-time sentinel")
    );
}
