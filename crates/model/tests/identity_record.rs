// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{FileIdentity, FileIdentityRecord, RecipeDigest};
use serde_json::{Value, json};

const GUID: &str = "11111111-2222-3333-4444-555555555555";

fn fresh_identity() -> FileIdentity {
    FileIdentity {
        internal_name: GUID.parse().unwrap(),
        revision_id: None,
        database_revision_id: None,
        save_counter: 0,
        version_created: "alpha".into(),
        version_saved: String::new(),
        model_digest: None,
    }
}

fn saved_mapping() -> Value {
    json!({
        "internalName": GUID,
        "revisionId": "00000000-0000-4000-8000-000000000002",
        "databaseRevisionId": "00000000-0000-4000-8000-000000000003",
        "saveCounter": 4294967296_u64,
        "versionCreated": "alpha",
        "versionSaved": "beta+build.4",
        "modelDigest": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    })
}

#[test]
fn saved_record_round_trip_preserves_every_legacy_field() {
    let mapping = saved_mapping();
    let record: FileIdentityRecord = serde_json::from_value(mapping.clone()).unwrap();
    let identity = record.restore(&fresh_identity()).unwrap();
    assert_eq!(identity.save_counter, 4294967296);
    assert_eq!(identity.model_digest, Some(RecipeDigest::of_recipe(b"abc")));
    assert_eq!(identity.version_saved, "beta+build.4");
    let restored = FileIdentityRecord::from(&identity);
    assert_eq!(serde_json::to_value(&restored).unwrap(), mapping);
    assert_eq!(restored.restore(&fresh_identity()).unwrap(), identity);
}

#[test]
fn unsaved_record_omits_empty_legacy_fields() {
    let identity = fresh_identity();
    let record = FileIdentityRecord::from(&identity);
    let expected = json!({"internalName": GUID, "versionCreated": "alpha"});
    assert_eq!(serde_json::to_value(&record).unwrap(), expected);
    let decoded: FileIdentityRecord = serde_json::from_value(expected).unwrap();
    assert_eq!(decoded.restore(&identity).unwrap(), identity);
    assert_eq!(
        serde_json::to_value(FileIdentityRecord::default()).unwrap(),
        json!({})
    );
}

#[test]
fn pre_identity_files_retain_the_already_allocated_identity() {
    let fresh = fresh_identity();
    for mapping in [
        json!({}),
        json!({"revisionId": "ignored without internalName"}),
    ] {
        let record: FileIdentityRecord = serde_json::from_value(mapping).unwrap();
        assert_eq!(record.restore(&fresh).unwrap(), fresh);
    }
}

#[test]
fn malformed_guid_fields_report_the_field_and_value() {
    for field in ["internalName", "revisionId", "databaseRevisionId"] {
        let mut mapping = saved_mapping();
        mapping[field] = json!("invalid-guid");
        let record: FileIdentityRecord = serde_json::from_value(mapping).unwrap();
        let error = record.restore(&fresh_identity()).unwrap_err();
        assert!(error.to_string().contains(field));
        assert!(error.to_string().contains("invalid-guid"));
        assert!(error.to_string().contains("expected 32 hexadecimal digits"));
        assert!(std::error::Error::source(&error).is_none());
    }
}

#[test]
fn digest_requires_exactly_64_ascii_hex_digits() {
    for invalid in [
        "abc".to_owned(),
        "g".repeat(64),
        "é".repeat(32),
        "a".repeat(66),
    ] {
        let mut mapping = saved_mapping();
        mapping["modelDigest"] = json!(invalid);
        let record: FileIdentityRecord = serde_json::from_value(mapping).unwrap();
        let error = record.restore(&fresh_identity()).unwrap_err();
        assert!(error.to_string().contains("modelDigest"));
        assert!(error.to_string().contains(&invalid));
        assert!(
            error
                .to_string()
                .contains("expected 64 SHA-256 hexadecimal digits")
        );
    }
}

#[test]
fn accepted_hex_spelling_normalizes_without_changing_identity() {
    let mut mapping = saved_mapping();
    mapping["internalName"] = json!("AAAAAAAABBBBCCCCDDDDEEEEEEEEEEEE");
    let digest = mapping["modelDigest"].as_str().unwrap().to_uppercase();
    mapping["modelDigest"] = json!(digest);
    let record: FileIdentityRecord = serde_json::from_value(mapping).unwrap();
    let identity = record.restore(&fresh_identity()).unwrap();
    assert_eq!(
        identity.internal_name.to_string(),
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    );
    assert_eq!(identity.model_digest, Some(RecipeDigest::of_recipe(b"abc")));
}

#[test]
fn unsupported_fields_and_invalid_scalar_shapes_are_refused() {
    for mapping in [
        json!({"saveCounter": -1}),
        json!({"saveCounter": 1.5}),
        json!({"saveCounter": "1"}),
        json!({"internalName": null}),
        json!({"futureIdentity": "do not silently discard"}),
    ] {
        assert!(serde_json::from_value::<FileIdentityRecord>(mapping).is_err());
    }
    let mapping = json!({"internalName": GUID, "saveCounter": u64::MAX});
    let record: FileIdentityRecord = serde_json::from_value(mapping).unwrap();
    assert_eq!(
        record.restore(&fresh_identity()).unwrap().save_counter,
        u64::MAX
    );
}
