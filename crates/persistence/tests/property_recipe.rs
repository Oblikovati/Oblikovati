// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{
    json::Bytes,
    types::{USER_PARAM, VariantValue},
};
use oblikovati_model::{parameter::ParameterCollection, property::*, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

fn package(records: &str) -> DocumentPackage {
    DocumentPackage::decode(format!("schemaVersion: 2\ndocumentType: 1\ndata: {{addin: YWJj}}\nmodel: {{properties: {records}, extension: retained}}\n").as_bytes()).unwrap()
}

#[test]
fn every_typed_value_and_insertion_order_survive_export_and_reopen() {
    let mut properties = DocumentProperties::default();
    for (name, value) in [
        ("flag", VariantValue::Boolean(false)),
        ("min", VariantValue::Integer(i64::MIN)),
        ("max", VariantValue::Integer(i64::MAX)),
        ("zero", VariantValue::Double(-0.0)),
        ("small", VariantValue::Double(f64::from_bits(1))),
        ("large", VariantValue::Double(f64::MAX)),
        ("text", VariantValue::String("a: 日本語\nline".into())),
        ("bytes", VariantValue::Bytes(Bytes(Some(vec![0, 255])))),
        ("empty", VariantValue::Bytes(Bytes(None))),
    ] {
        properties.put("Custom", name, value).unwrap();
    }
    let original = package("[]");
    let exported = original.with_properties(&properties).unwrap();
    let reopened = DocumentPackage::decode(&exported.encode().unwrap()).unwrap();
    assert_eq!(reopened.restore_properties().unwrap(), properties);
    let VariantValue::Double(zero) = reopened
        .restore_properties()
        .unwrap()
        .get("Custom", "zero")
        .unwrap()
        .value
    else {
        panic!("expected double");
    };
    assert_eq!(zero.to_bits(), (-0.0_f64).to_bits());
    assert_eq!(exported.streams(), original.streams());
    assert!(
        String::from_utf8(exported.section_yaml("model").unwrap().unwrap())
            .unwrap()
            .contains("extension: retained")
    );
    assert_eq!(original, package("[]"));
    assert_eq!(exported.with_properties(&properties).unwrap(), exported);
}

#[test]
fn go_generated_fixture_restores_all_value_types() {
    let recipe = include_str!("fixtures/go-document-properties.yaml")
        .lines()
        .map(|line| format!("  {line}\n"))
        .collect::<String>();
    let original = DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n{recipe}").as_bytes(),
    )
    .unwrap();
    let properties = original.restore_properties().unwrap();
    assert_eq!(
        properties.get(SUMMARY_INFORMATION, "Title").unwrap().value,
        VariantValue::String("Steel: 日本語".into())
    );
    assert_eq!(
        properties.get(USER_DEFINED, "Approved").unwrap().value,
        VariantValue::Boolean(true)
    );
    assert_eq!(
        properties.get(USER_DEFINED, "Count").unwrap().value,
        VariantValue::Integer(i64::MAX)
    );
    assert_eq!(
        properties.get(USER_DEFINED, "Cost").unwrap().value,
        VariantValue::Double(12.125)
    );
    assert_eq!(
        properties.get(USER_DEFINED, "Payload").unwrap().value,
        VariantValue::Bytes(Bytes(Some(vec![0, 255, 1])))
    );
    assert_eq!(
        properties.get(USER_DEFINED, "Empty").unwrap().value,
        VariantValue::Bytes(Bytes(None))
    );
    assert_eq!(
        original
            .with_properties(&properties)
            .unwrap()
            .restore_properties()
            .unwrap(),
        properties
    );
}

#[test]
fn go_boolean_aliases_and_base64_rules_are_preserved() {
    for (value, expected) in [
        ("1", true),
        ("t", true),
        ("T", true),
        ("true", true),
        ("TRUE", true),
        ("True", true),
        ("0", false),
        ("f", false),
        ("F", false),
        ("false", false),
        ("FALSE", false),
        ("False", false),
    ] {
        let original = package(&format!(
            "[{{set: Custom, name: flag, type: boolean, value: '{value}'}}]"
        ));
        assert_eq!(
            original
                .restore_properties()
                .unwrap()
                .get("Custom", "flag")
                .unwrap()
                .value,
            VariantValue::Boolean(expected)
        );
    }
    let original = package("[{set: Custom, name: bytes, type: bytes, value: \"Zh==\\r\\n\"}]");
    assert_eq!(
        original
            .restore_properties()
            .unwrap()
            .get("Custom", "bytes")
            .unwrap()
            .value,
        VariantValue::Bytes(Bytes(Some(vec![b'f'])))
    );
}

#[test]
fn invalid_or_duplicate_property_records_refuse_load_and_replacement() {
    for record in [
        "{}",
        "[{}]",
        "[{set: Custom, name: x, type: future, value: ''}]",
        "[{set: Custom, name: x, type: integer, value: '9223372036854775808'}]",
        "[{set: Custom, name: x, type: boolean, value: yes}]",
        "[{set: Custom, name: x, type: double, value: NaN}]",
        "[{set: Custom, name: x, type: bytes, value: 'invalid'}]",
        "[{set: Custom, name: '', type: string, value: x}]",
        "[{set: Custom, name: x, type: string, value: x, future: true}]",
        "[{set: Custom, name: x, type: string, value: first}, {set: Custom, name: x, type: string, value: second}]",
    ] {
        let original = package(record);
        assert!(original.restore_properties().is_err(), "{record}");
        assert!(
            original
                .with_properties(&DocumentProperties::default())
                .is_err(),
            "{record}"
        );
    }
}

#[test]
fn empty_custom_sets_promote_and_standalone_export_still_refuses_provenance_loss() {
    let mut properties = DocumentProperties::default();
    properties
        .put("Custom", "temporary", VariantValue::Boolean(true))
        .unwrap();
    properties.remove("Custom", "temporary");
    let saved = package("[]").with_properties(&properties).unwrap();
    assert_eq!(saved.manifest().schema_version, 3);
    assert_eq!(saved.restore_properties().unwrap(), properties);

    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1")
        .unwrap();
    properties.expose_parameter(&parameters, 1).unwrap();
    assert!(
        package("[]")
            .with_properties(&properties)
            .unwrap_err()
            .to_string()
            .contains("parameter exposure")
    );
}

#[test]
fn absent_null_and_empty_recipes_recreate_standard_sets() {
    for records in ["null", "[]"] {
        assert_eq!(
            package(records).restore_properties().unwrap(),
            DocumentProperties::default()
        );
    }
    let absent = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert_eq!(
        absent.restore_properties().unwrap(),
        DocumentProperties::default()
    );
}
