// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

fn recipe(schema: u32, ids: &str, mark: &str) -> DocumentPackage {
    DocumentPackage::decode(format!("schemaVersion: {schema}\ndocumentType: 1\nmodel:\n  parameters: [{{name: width, kind: user, expression: '2 cm'}}, {{name: twice, kind: user, expression: 'width * 2'}}]\n  parameterIds: {ids}\n  parameterHighWater: {mark}\n").as_bytes()).unwrap()
}

#[test]
fn saved_ids_survive_order_changes_and_deleted_reservations() {
    let package = recipe(3, "{width: 7, twice: 2}", "19");
    let mut restored = package.restore_document_parameters().unwrap();
    assert_eq!(restored.parameters.ids(), [7, 2]);
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        4.0
    );
    assert_eq!(restored.ids.allocate().unwrap(), 20);
    restored.parameters.rename(7, "span").unwrap();
    let saved = package
        .with_parameters(&restored.parameters, UnitsOfMeasure::default())
        .unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap())
        .unwrap()
        .restore_document_parameters()
        .unwrap();
    assert_eq!(reopened.parameters.find("span"), Some(7));
    assert_eq!(reopened.ids.high_water(), 19);
}

#[test]
fn mapping_must_cover_exactly_the_live_names_with_unique_reserved_ids() {
    for (schema, ids, mark) in [
        (2, "{width: 7, twice: 2}", "9"),
        (3, "{width: 7}", "9"),
        (3, "{width: 7, twice: 2, extra: 8}", "9"),
        (3, "{width: 7, twice: 7}", "9"),
        (3, "{width: 0, twice: 2}", "9"),
        (3, "{width: 7, twice: 2}", "6"),
        (3, "{width: 7, twice: 2}", "-1"),
        (3, "{width: 7, twice: 2}", "null"),
    ] {
        assert!(
            recipe(schema, ids, mark)
                .restore_document_parameters()
                .is_err(),
            "{schema}: {ids}, {mark}"
        );
    }
}

#[test]
fn empty_live_set_can_keep_a_full_reservation_counter() {
    let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Empty");
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    assert_eq!(
        package.with_parameter_reservations(&parameters, 0).unwrap(),
        package
    );
    let saved = package
        .with_parameter_reservations(&parameters, u64::MAX)
        .unwrap();
    let mut restored = saved.restore_document_parameters().unwrap();
    assert!(restored.parameters.ids().is_empty());
    assert!(restored.ids.allocate().is_err());
    assert!(saved.with_parameter_reservations(&parameters, 0).is_err());
}

#[test]
fn direct_capture_preserves_nonsequential_ids_when_promoted() {
    let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Part");
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(9, "length", USER_PARAM, "3 cm")
        .unwrap();
    let saved = package
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap()
        .with_parameter_reservations(&parameters, 12)
        .unwrap();
    assert!(
        saved
            .with_parameter_reservations(&ParameterCollection::new(UnitsOfMeasure::default()), 0)
            .is_err()
    );
    let restored = saved.restore_document_parameters().unwrap();
    assert_eq!(restored.parameters.find("length"), Some(9));
    assert_eq!(restored.ids.high_water(), 12);
    assert!(saved.with_parameter_reservations(&parameters, 8).is_err());
}

#[test]
fn partial_identity_metadata_is_refused() {
    for field in ["parameterIds: {}", "parameterHighWater: 1"] {
        let package = DocumentPackage::decode(
            format!("schemaVersion: 3\ndocumentType: 1\nmodel:\n  {field}\n").as_bytes(),
        )
        .unwrap();
        let error = package
            .restore_document_parameters()
            .unwrap_err()
            .to_string();
        assert!(
            error.contains("both parameterIds and parameterHighWater"),
            "{error}"
        );
    }
}
