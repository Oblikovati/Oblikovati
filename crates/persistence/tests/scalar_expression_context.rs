// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

fn input_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_scalar_expression(1, "depth", " 10 ", Dimension::Length)
        .unwrap();
    let mut preferences = parameters.units().clone();
    preferences.set_preferred(Dimension::Length, "in").unwrap();
    parameters.set_unit_preferences(preferences).unwrap();
    parameters
}
fn save(parameters: &ParameterCollection<u64>) -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(parameters, parameters.units().clone())
        .unwrap()
}
fn roundtrip(parameters: &ParameterCollection<u64>) -> ParameterCollection<u64> {
    DocumentPackage::decode(&save(parameters).encode().unwrap())
        .unwrap()
        .restore_document_parameters()
        .unwrap()
        .parameters
}

#[test]
fn reopened_bare_expression_and_user_copy_keep_captured_unit_and_source() {
    let mut parameters = input_parameters();
    parameters.copy_to_user(1, 2).unwrap();
    let restored = roundtrip(&parameters);
    for id in [1, 2] {
        assert_eq!(restored.get(id).unwrap().source, " 10 ");
        assert_eq!(restored.model_value(id).unwrap().value, 1.0);
        assert_eq!(
            restored.get(id).unwrap().expression_context,
            parameters.get(id).unwrap().expression_context
        );
    }
    assert_eq!(restored.units().preferred_name(Dimension::Length), "in");
}

#[test]
fn context_survives_measured_constant_replacement_and_reopen() {
    let mut parameters = input_parameters();
    parameters
        .set_value(1, Quantity::new(3.25, Dimension::Length))
        .unwrap();
    let mut restored = roundtrip(&parameters);
    assert_eq!(restored.model_value(1).unwrap().value, 3.25);
    assert_eq!(restored.get(1).unwrap().source, "3.25 length");
    restored.set_expression(1, "2").unwrap();
    assert_eq!(restored.model_value(1).unwrap().value, 5.08);
}

#[test]
fn context_requires_schema_three_and_unknown_fields_are_refused() {
    let encoded = String::from_utf8(save(&input_parameters()).encode().unwrap()).unwrap();
    assert!(encoded.contains("schemaVersion: 3"));
    let old_schema = encoded.replace("schemaVersion: 3", "schemaVersion: 2");
    let package = DocumentPackage::decode(old_schema.as_bytes()).unwrap();
    let error = package.restore_document_parameters().err().unwrap();
    assert!(
        error
            .to_string()
            .contains("schema 2: expected document schema 3")
    );
    let invalid = encoded.replace("bare_unit: mm", "unexpected: mm");
    assert_ne!(invalid, encoded);
    assert!(
        DocumentPackage::decode(invalid.as_bytes())
            .unwrap()
            .restore_document_parameters()
            .is_err()
    );
}

#[test]
fn invalid_stored_context_is_refused_before_parameter_restore() {
    let encoded = String::from_utf8(save(&input_parameters()).encode().unwrap()).unwrap();
    for (original, replacement) in [
        ("version: 1", "version: 2"),
        ("bare_unit: mm", "bare_unit: deg"),
        ("category: length", "category: text"),
    ] {
        let invalid = encoded.replace(original, replacement);
        assert_ne!(invalid, encoded);
        let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
        assert!(package.restore_document_parameters().is_err());
    }
}

#[test]
fn failed_context_formula_restores_health_without_changing_authored_source() {
    let mut parameters = input_parameters();
    parameters
        .set_scalar_expression(1, "missing / 2", Dimension::Length)
        .unwrap();
    let restored = roundtrip(&parameters);
    assert_eq!(restored.get(1).unwrap().source, "missing / 2");
    assert!(restored.model_value(1).is_err());
    assert_eq!(
        restored.get(1).unwrap().last_good_value,
        parameters.get(1).unwrap().last_good_value
    );
}
