// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DocumentInterestRecord, INTEREST_NONE, INTERESTED};
use oblikovati_model::interest::DocumentInterests;
use oblikovati_persistence::package::DocumentPackage;

fn package(section: &str) -> DocumentPackage {
    DocumentPackage::decode(format!("schemaVersion: 2\ndocumentType: 12290\n{section}").as_bytes())
        .unwrap()
}

#[test]
fn go_interest_shape_restores_and_round_trips_without_changing_other_sections() {
    let original = DocumentPackage::decode(include_bytes!("fixtures/go-interests.yml")).unwrap();
    let mut interests = original.restore_interests().unwrap();
    assert_eq!(interests.records().len(), 2);
    assert_eq!(interests.records()[0].interest_type, INTERESTED);
    assert_eq!(interests.records()[0].data_version, 3);
    assert_eq!(interests.records()[1].interest_type, INTEREST_NONE);
    assert_eq!(interests.records()[1].client_data, "µm\0\n");
    interests
        .put(DocumentInterestRecord {
            client_id: "cam".into(),
            name: "paths".into(),
            data_version: i64::MAX,
            client_data: "replacement".into(),
            ..Default::default()
        })
        .unwrap();
    let captured = original.with_interests(&interests).unwrap();
    assert_eq!(captured.restore_interests().unwrap(), interests);
    let restored = DocumentPackage::decode(&captured.encode().unwrap()).unwrap();
    assert_eq!(restored.restore_interests().unwrap(), interests);
    for section in ["model", "data", "attributes"] {
        assert_eq!(
            original.section_yaml(section).unwrap(),
            restored.section_yaml(section).unwrap()
        );
    }
    assert_eq!(
        original.recipe_digest().unwrap(),
        restored.recipe_digest().unwrap()
    );
    let cleared = restored
        .with_interests(&DocumentInterests::default())
        .unwrap();
    assert!(cleared.section_yaml("interests").unwrap().is_none());
    assert!(cleared.restore_interests().unwrap().records().is_empty());
}

#[test]
fn absent_empty_and_default_strength_are_supported() {
    for section in ["", "interests: null", "interests: []"] {
        assert!(
            package(section)
                .restore_interests()
                .unwrap()
                .records()
                .is_empty()
        );
    }
    let restored = package("interests: [{clientId: client, name: name}]")
        .restore_interests()
        .unwrap();
    assert_eq!(restored.records()[0].interest_type, INTERESTED);
    assert_eq!(restored.records()[0].data_version, 0);
}

#[test]
fn invalid_records_cannot_be_silently_overwritten() {
    for record in [
        "{clientId: '', name: x}",
        "{clientId: x, name: ''}",
        "{clientId: x}",
        "{clientId: x, name: y, interestType: future}",
        "{clientId: x, name: y, interestType: 68866}",
        "{clientId: x, name: y, extra: retained}",
        "{clientId: x, name: y, dataVersion: 9223372036854775808}",
        "{clientId: x, name: y, dataVersion: 1.2}",
        "{clientId: x, name: y}, {clientId: x, name: y}",
    ] {
        let original = package(&format!("interests: [{record}]"));
        assert!(original.restore_interests().is_err(), "{record}");
        assert!(
            original
                .with_interests(&DocumentInterests::default())
                .is_err(),
            "{record}"
        );
    }
}
