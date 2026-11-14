// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

#[test]
fn built_in_model_parameter_survives_save_and_keeps_its_conversion_guard() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "size", USER_PARAM, "10 mm")
        .unwrap();
    parameters.insert_auto_model(2, "size").unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let saved = package
        .with_parameters(&parameters, parameters.units().clone())
        .unwrap();
    assert_eq!(saved.manifest().schema_version, 3);
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let mut restored = reopened.restore_document_parameters().unwrap().parameters;
    assert_eq!(restored.get(2), parameters.get(2));
    assert!(restored.get(2).unwrap().built_in);
    assert!(restored.convert_kind(2, USER_PARAM).is_err());
    restored.set_expression(1, "20 mm").unwrap();
    assert_eq!(restored.get(2).unwrap().current_value().unwrap().value, 2.0);
    assert_eq!(restored.get(2).unwrap().kind, MODEL_PARAM);
}

#[test]
fn invalid_or_unversioned_builtin_flags_are_refused() {
    for (schema, row) in [
        (
            2,
            "{name: d0, kind: model, expression: '1 cm', builtIn: true}",
        ),
        (
            3,
            "{name: d0, kind: user, expression: '1 cm', builtIn: true}",
        ),
        (
            3,
            "{name: d0, kind: model, expression: '1 cm', builtIn: false}",
        ),
        (
            3,
            "{name: d0, kind: model, expression: '1 cm', builtIn: invalid}",
        ),
        (
            3,
            "{name: d0, kind: model, expression: '1 cm', builtIn: null}",
        ),
    ] {
        let input =
            format!("schemaVersion: {schema}\ndocumentType: 1\nmodel:\n  parameters: [{row}]\n");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(package.restore_document_parameters().is_err(), "{input}");
    }
}

#[test]
fn failed_builtin_dependency_keeps_the_guard_and_failure_after_reopen() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "size", USER_PARAM, "10 mm")
        .unwrap();
    parameters.insert_auto_model(2, "size").unwrap();
    parameters.set_expression(1, "missing").unwrap();
    assert!(parameters.get(2).unwrap().current_value().is_none());
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n").unwrap();
    let saved = package
        .with_parameters(&parameters, parameters.units().clone())
        .unwrap();
    let mut restored = saved.restore_document_parameters().unwrap().parameters;
    assert_eq!(restored.get(2), parameters.get(2));
    assert!(restored.convert_kind(2, USER_PARAM).is_err());
    restored.set_expression(1, "20 mm").unwrap();
    assert_eq!(restored.get(2).unwrap().current_value().unwrap().value, 2.0);
}
