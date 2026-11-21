// SPDX-License-Identifier: GPL-2.0-only
//! Property provenance and invalid attribute load checks.
use super::*;

#[test]
fn loaded_parameter_export_flags_rebuild_property_provenance() {
    use oblikovati_api::types::VariantValue;
    use oblikovati_model::property::USER_DEFINED;
    use oblikovati_persistence::load::load_parameter_package;
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: [{name: label, kind: user, valueType: text, text: steel, export: true}]}\n";
    let loaded = load_parameter_package(
        Path::new("exposed.opd"),
        1,
        "alpha",
        &mut DocumentIdentityIndex::default(),
        &mut source(),
        &mut reader(encoded),
    )
    .unwrap();
    let property = loaded.properties.get(USER_DEFINED, "label").unwrap();
    assert_eq!(property.value, VariantValue::String("steel".into()));
    assert_eq!(property.from_parameter, "label");
}

#[test]
fn malformed_attributes_claim_no_document_identity() {
    use oblikovati_persistence::load::load_parameter_package;
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nattributes: [1, 0, 0, 0]\n";
    let result = load_parameter_package(
        Path::new("attributes.opd"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut reader(encoded),
    );
    assert!(result.unwrap_err().to_string().contains("attributes"));
    assert!(index.identity(1).is_none());
    assert_eq!(generator.next, 0);
}

#[test]
fn document_properties_are_reconstructed_before_registration() {
    use oblikovati_api::types::VariantValue;
    use oblikovati_persistence::load::load_parameter_package;
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    for (property_type, expected_ok) in [("unknown", false), ("integer", true)] {
        let encoded = format!(
            "schemaVersion: 2\ndocumentType: 1\nmodel: {{properties: [{{set: Custom, name: Count, type: {property_type}, value: '4'}}]}}\n"
        );
        let result = load_parameter_package(
            Path::new("properties.opd"),
            1,
            "alpha",
            &mut index,
            &mut generator,
            &mut reader(encoded.as_bytes()),
        );
        if !expected_ok {
            assert!(result.is_err());
            assert!(index.identity(1).is_none());
            assert_eq!(generator.next, 0);
            continue;
        }
        assert_eq!(
            result
                .unwrap()
                .properties
                .get("Custom", "Count")
                .unwrap()
                .value,
            VariantValue::Integer(4)
        );
        assert!(index.identity(1).is_some());
    }
}
