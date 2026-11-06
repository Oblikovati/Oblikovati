// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{
    DocumentInterestRecord, DocumentInterestType, INTEREST_NONE, INTERESTED,
};
use oblikovati_model::interest::DocumentInterests;

fn record(client: &str, name: &str) -> DocumentInterestRecord {
    DocumentInterestRecord {
        client_id: client.into(),
        name: name.into(),
        ..Default::default()
    }
}

#[test]
fn identity_update_retains_order_and_discovery_includes_withdrawn_records() {
    let mut interests = DocumentInterests::default();
    assert!(!interests.has_interest(""));
    for (client, name) in [("cam", "path"), ("cam", "setup"), ("other", "path")] {
        interests.put(record(client, name)).unwrap();
    }
    assert_eq!(interests.records()[0].interest_type, INTERESTED);
    let replacement = DocumentInterestRecord {
        interest_type: INTEREST_NONE,
        data_version: i64::MIN,
        client_data: "µm\0\n{opaque: true}".into(),
        ..record("cam", "path")
    };
    interests.put(replacement.clone()).unwrap();
    assert_eq!(interests.records().len(), 3);
    assert_eq!(interests.records()[0], replacement);
    assert!(interests.has_interest("cam"));
    assert!(interests.has_interest("path"));
    assert!(!interests.has_interest("CAM"));
    assert!(interests.remove("cam", "path"));
    assert!(!interests.remove("cam", "path"));
    assert_eq!(interests.records()[0].name, "setup");
    assert_eq!(interests.records()[1].client_id, "other");
}

#[test]
fn rejected_records_leave_collection_unchanged_and_copies_are_independent() {
    let mut interests = DocumentInterests::default();
    interests.put(record("client", "name")).unwrap();
    let original = interests.clone();
    for invalid in [
        record("", "name"),
        record("client", ""),
        DocumentInterestRecord {
            interest_type: DocumentInterestType(7),
            ..record("client", "name")
        },
    ] {
        let error = interests.put(invalid).unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
        assert_eq!(interests, original);
    }
    let mut copied = interests.clone();
    copied.remove("client", "name");
    assert!(copied.records().is_empty());
    assert_eq!(interests, original);
    interests.put(record("client", "name")).unwrap();
    assert_eq!(interests, original);
}
