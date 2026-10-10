// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::DocumentGuid;
use std::collections::BTreeSet;

const DOCUMENT_GUID: &str = "11111111-2222-3333-4444-555555555555";

const MALFORMED_GUIDS: [&str; 8] = [
    "",
    "not-a-uuid",
    "11111111-2222-3333-4444",
    "1111111122223333444455555555555g",
    "-11111111222233334444555555555555---",
    "{11111111-2222-3333-4444-555555555555}",
    "urn:uuid:11111111-2222-3333-4444-555555555555",
    "é1111111222233334444555555555555",
];

#[test]
fn derived_references_match_saved_go_values() {
    let fixture = include_str!("fixtures/go-sketch-identity.tsv");
    let rows: Vec<&str> = fixture
        .lines()
        .filter(|row| !row.starts_with('#'))
        .collect();
    assert_eq!(rows.len(), 15);
    for row in rows {
        let columns: Vec<&str> = row.split('\t').collect();
        assert_eq!(columns.len(), 4);
        let document: DocumentGuid = columns[0].parse().unwrap();
        let local_id: u64 = columns[1].parse().unwrap();
        assert_eq!(document.sketch_key(local_id).to_string(), columns[2]);
        assert_eq!(document.sketch_entity_key(local_id).to_string(), columns[3]);
        assert_eq!(document.to_string(), columns[0]);
    }
}

#[test]
fn guid_spelling_does_not_change_the_namespace() {
    let canonical: DocumentGuid = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    for source in [
        "AAAAAAAABBBBCCCCDDDDEEEEEEEEEEEE",
        "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
    ] {
        let parsed: DocumentGuid = source.parse().unwrap();
        assert_eq!(parsed, canonical);
        assert_eq!(parsed.to_string(), canonical.to_string());
        assert_eq!(
            parsed.sketch_entity_key(42),
            canonical.sketch_entity_key(42)
        );
    }
}

#[test]
fn malformed_namespaces_report_the_input_and_expected_shape() {
    for source in MALFORMED_GUIDS {
        let error = source.parse::<DocumentGuid>().unwrap_err();
        assert!(error.to_string().contains(&format!("{source:?}")));
        assert!(
            error
                .to_string()
                .contains("expected 32 hexadecimal digits or an 8-4-4-4-12 GUID")
        );
        assert!(std::error::Error::source(&error).is_none());
    }
}

#[test]
fn document_kind_and_local_id_separate_reference_names() {
    let documents: [DocumentGuid; 2] = [
        DOCUMENT_GUID.parse().unwrap(),
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
    ];
    let mut references = BTreeSet::new();
    for document in documents {
        for local_id in 0..256 {
            assert!(references.insert(document.sketch_key(local_id)));
            assert!(references.insert(document.sketch_entity_key(local_id)));
            assert_eq!(document.sketch_key(local_id), document.sketch_key(local_id));
        }
    }
    assert_eq!(references.len(), 1024);
}
