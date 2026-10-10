// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DOCUMENT_PART, USER_PARAM, VariantValue};
use oblikovati_model::{
    parameter::ParameterCollection,
    property::{DocumentProperties, USER_DEFINED},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

#[test]
fn empty_and_nonempty_custom_sets_keep_their_order_across_round_trips() {
    let mut properties = DocumentProperties::default();
    properties.ensure_set("Empty first").unwrap();
    properties
        .put("Filled", "value", VariantValue::Integer(7))
        .unwrap();
    properties.ensure_set("Empty last").unwrap();
    let original = DocumentPackage::empty(DOCUMENT_PART, "Part");
    let mut saved = original.with_properties(&properties).unwrap();
    for _ in 0..3 {
        saved = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
        assert_eq!(saved.restore_properties().unwrap(), properties);
        saved = saved.with_properties(&properties).unwrap();
    }
    assert_eq!(original.manifest().schema_version, 2);
    let cleared = saved
        .with_properties(&DocumentProperties::default())
        .unwrap();
    assert_eq!(
        cleared.restore_properties().unwrap(),
        DocumentProperties::default()
    );
}

#[test]
fn declaration_only_packages_restore_empty_sets_and_reject_invalid_membership() {
    let valid = DocumentPackage::decode(
        b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  customPropertySets: [First, Second]\n",
    )
    .unwrap();
    assert_eq!(valid.restore_properties().unwrap().sets().len(), 6);
    for (schema, names, records) in [
        (2, "[Empty]", "[]"),
        (3, "['']", "[]"),
        (3, "[Same, Same]", "[]"),
        (3, "['User Defined Properties']", "[]"),
        (3, "null", "[]"),
        (
            3,
            "[]",
            "[{set: Missing, name: value, type: string, value: text}]",
        ),
    ] {
        let package = DocumentPackage::decode(format!("schemaVersion: {schema}\ndocumentType: 1\nmodel:\n  customPropertySets: {names}\n  properties: {records}\n").as_bytes()).unwrap();
        assert!(
            package.restore_properties().is_err(),
            "{schema} {names} {records}"
        );
    }
}

#[test]
fn exposed_parameter_capture_preserves_empty_sets_and_parameter_ids() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(7, "width", USER_PARAM, "4 cm")
        .unwrap();
    let mut annotation = parameters.get(7).unwrap().annotations.clone();
    annotation.exposed_as_property = true;
    parameters.set_annotations(7, annotation).unwrap();
    let mut properties = DocumentProperties::default();
    properties.ensure_set("Reserved").unwrap();
    let saved = DocumentPackage::empty(DOCUMENT_PART, "Part")
        .with_parameter_properties(&parameters, &properties, UnitsOfMeasure::default())
        .unwrap();
    let restored = saved.restore_document_parameters().unwrap();
    assert_eq!(restored.parameters.find("width"), Some(7));
    let properties = saved
        .restore_parameter_properties(&restored.parameters)
        .unwrap();
    assert!(properties.set("Reserved").unwrap().properties().is_empty());
    assert_eq!(
        properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .from_parameter,
        "width"
    );
}
