// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DOCUMENT_PART, USER_PARAM, VariantValue};
use oblikovati_model::{
    attribute::{AttributeTarget, DocumentAttributes},
    parameter::ParameterCollection,
    property::{DocumentProperties, USER_DEFINED},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::{metadata::decode_attributes, package::DocumentPackage};

#[test]
fn packaged_go_attributes_survive_property_and_parameter_updates() {
    let fixture = include_bytes!("fixtures/go-binary-attributes.bin");
    let attributes = decode_attributes(fixture).unwrap();
    let original = DocumentPackage::empty(DOCUMENT_PART, "Part");
    let saved = original.with_attributes(&attributes).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    assert!(
        reopened
            .restore_attributes()
            .unwrap()
            .same_recipe(&attributes)
    );
    assert!(original.restore_attributes().unwrap().anchors().is_empty());
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "Label", VariantValue::String("Part".into()))
        .unwrap();
    let edited = reopened
        .with_parameter_properties(&parameters, &properties, UnitsOfMeasure::default())
        .unwrap();
    assert!(
        edited
            .restore_attributes()
            .unwrap()
            .same_recipe(&attributes)
    );
    let cleared = edited
        .with_attributes(&DocumentAttributes::default())
        .unwrap();
    assert!(
        !String::from_utf8(cleared.encode().unwrap())
            .unwrap()
            .contains("attributes:")
    );
    assert_eq!(cleared.restore_properties().unwrap(), properties);
    assert_eq!(
        cleared
            .restore_document_parameters()
            .unwrap()
            .parameters
            .find("width"),
        Some(1)
    );
}

#[test]
fn empty_anchors_and_signed_zero_remain_exact_across_yaml_reconstruction() {
    let mut attributes = DocumentAttributes::default();
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    attributes.ensure_target(&target);
    let package = DocumentPackage::empty(DOCUMENT_PART, "Part")
        .with_attributes(&attributes)
        .unwrap();
    assert!(
        package
            .restore_attributes()
            .unwrap()
            .same_recipe(&attributes)
    );
    attributes
        .put(&target, "AddIn", "zero", VariantValue::Double(-0.0))
        .unwrap();
    let package = package.with_attributes(&attributes).unwrap();
    let restored = DocumentPackage::decode(&package.encode().unwrap())
        .unwrap()
        .restore_attributes()
        .unwrap();
    assert!(restored.same_recipe(&attributes));
}

#[test]
fn malformed_sections_refuse_restore_and_replacement() {
    for section in [
        "[256]",
        "[-1]",
        "[1.5]",
        "[true]",
        "[text]",
        "[1, 0, 0, 0]",
        "[0, 0, 0, 0, 1]",
    ] {
        let package = DocumentPackage::decode(
            format!("schemaVersion: 2\ndocumentType: 1\nattributes: {section}\n").as_bytes(),
        )
        .unwrap();
        assert!(package.restore_attributes().is_err(), "{section}");
        assert!(
            package
                .with_attributes(&DocumentAttributes::default())
                .is_err(),
            "{section}"
        );
    }
    for section in ["null", "[]", "[0, 0, 0, 0]"] {
        let package = DocumentPackage::decode(
            format!("schemaVersion: 2\ndocumentType: 1\nattributes: {section}\n").as_bytes(),
        )
        .unwrap();
        assert!(package.restore_attributes().unwrap().anchors().is_empty());
    }
}
