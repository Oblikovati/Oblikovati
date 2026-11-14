// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM, VariantValue};
use oblikovati_model::{
    parameter::ParameterCollection,
    property::{DocumentProperties, USER_DEFINED},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

fn exposed() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "25 mm")
        .unwrap();
    let mut annotations = parameters.get(1).unwrap().annotations.clone();
    annotations.exposed_as_property = true;
    parameters.set_annotations(1, annotations).unwrap();
    parameters
}

fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap()
}

#[test]
fn coupled_export_rebuilds_provenance_and_updates_selected_values() {
    let mut parameters = exposed();
    parameters.set_deviation(1, 0.1, -0.2).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "Author", VariantValue::String("Alice".into()))
        .unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    parameters.set_expression(1, "30 mm").unwrap();
    let updated = package()
        .with_parameter_properties(&parameters, &properties, UnitsOfMeasure::default())
        .unwrap();
    let reopened = DocumentPackage::decode(&updated.encode().unwrap()).unwrap();
    let restored = reopened.restore_document_parameters().unwrap();
    let output = reopened
        .restore_parameter_properties(&restored.parameters)
        .unwrap();
    assert_eq!(
        output.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::String("31 mm".into())
    );
    assert_eq!(
        output.get(USER_DEFINED, "width").unwrap().from_parameter,
        "width"
    );
    assert_eq!(
        output.get(USER_DEFINED, "Author"),
        properties.get(USER_DEFINED, "Author")
    );
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::String("26 mm".into())
    );
    assert!(
        reopened
            .restore_properties()
            .unwrap()
            .get(USER_DEFINED, "width")
            .unwrap()
            .from_parameter
            .is_empty()
    );
}

#[test]
fn rename_unexport_and_removal_prune_only_owned_projections() {
    let mut parameters = exposed();
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "manual", VariantValue::Integer(7))
        .unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    parameters.rename(1, "span").unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert!(properties.get(USER_DEFINED, "width").is_none());
    assert_eq!(
        properties.get(USER_DEFINED, "span").unwrap().from_parameter,
        "span"
    );
    let mut annotations = parameters.get(1).unwrap().annotations.clone();
    annotations.exposed_as_property = false;
    parameters.set_annotations(1, annotations).unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert!(properties.get(USER_DEFINED, "span").is_none());
    assert!(properties.get(USER_DEFINED, "manual").is_some());
    let cleared = package()
        .with_parameter_properties(&parameters, &properties, UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(cleared.restore_properties().unwrap(), properties);
}

#[test]
fn unhealthy_exposure_round_trips_without_stale_properties() {
    let mut parameters = exposed();
    let mut properties = DocumentProperties::default();
    properties.sync_exposed_parameters(&parameters).unwrap();
    parameters.set_expression(1, "30 mm").unwrap();
    parameters
        .insert_expression(2, "label", USER_PARAM, "missing")
        .unwrap();
    let mut annotations = parameters.get(2).unwrap().annotations.clone();
    annotations.exposed_as_property = true;
    parameters.set_annotations(2, annotations).unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::String("30 mm".into())
    );
    assert!(properties.get(USER_DEFINED, "label").is_none());
    let original = package();
    let saved = original
        .with_parameter_properties(&parameters, &properties, UnitsOfMeasure::default())
        .unwrap();
    let restored = saved.restore_document_parameters().unwrap();
    assert!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .is_none()
    );
    assert!(
        saved
            .restore_parameter_properties(&restored.parameters)
            .unwrap()
            .get(USER_DEFINED, "label")
            .is_none()
    );
    assert_eq!(original, package());
}

#[test]
fn saved_export_flag_reconstructs_a_missing_or_stale_numeric_property() {
    let original = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: user, expression: '25 mm', export: true}]\n  properties: [{set: User Defined Properties, name: width, type: double, value: '999'}]\n").unwrap();
    let restored = original.restore_document_parameters().unwrap();
    let properties = original
        .restore_parameter_properties(&restored.parameters)
        .unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::String("25 mm".into())
    );
    assert_eq!(
        properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .from_parameter,
        "width"
    );
    let empty_properties = package()
        .restore_parameter_properties(&restored.parameters)
        .unwrap();
    assert_eq!(empty_properties, properties);
}

#[test]
fn text_and_boolean_exposures_round_trip_as_their_api_value_types() {
    let mut parameters = exposed();
    parameters.insert_text(2, "label", "steel").unwrap();
    parameters.insert_boolean(3, "approved", true).unwrap();
    for id in [2, 3] {
        let mut annotations = parameters.get(id).unwrap().annotations.clone();
        annotations.exposed_as_property = true;
        parameters.set_annotations(id, annotations).unwrap();
    }
    let updated = package()
        .with_parameter_properties(
            &parameters,
            &DocumentProperties::default(),
            UnitsOfMeasure::default(),
        )
        .unwrap();
    let restored = updated.restore_document_parameters().unwrap();
    let properties = updated
        .restore_parameter_properties(&restored.parameters)
        .unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "label").unwrap().value,
        VariantValue::String("steel".into())
    );
    assert_eq!(
        properties.get(USER_DEFINED, "approved").unwrap().value,
        VariantValue::Boolean(true)
    );
    assert_eq!(
        properties
            .get(USER_DEFINED, "approved")
            .unwrap()
            .from_parameter,
        "approved"
    );
}

#[test]
fn failed_saved_parameter_does_not_restore_a_stale_property_as_current() {
    let original = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: user, expression: missing, export: true}]\n  properties: [{set: User Defined Properties, name: width, type: double, value: '999'}]\n").unwrap();
    assert!(original.restore_document_parameters().is_err());
    let mut parameters = exposed();
    parameters.set_expression(1, "missing").unwrap();
    let properties = original.restore_parameter_properties(&parameters).unwrap();
    assert!(properties.get(USER_DEFINED, "width").is_none());
}

#[test]
fn saved_text_formats_rebuild_the_same_projection_from_parameter_preferences() {
    use oblikovati_api::types::{
        PRECISION_EIGHTHS_FRACTIONAL, PRECISION_SECONDS_TWO_DECIMAL_ANGLE,
        PRECISION_TWO_DECIMAL_PLACES,
    };
    use oblikovati_model::parameter::CustomPropertyFormat;
    for (expression, units, precision, expected) in [
        ("0.5 cm", "cm", PRECISION_TWO_DECIMAL_PLACES, ".50 cm"),
        ("1.375 in", "in", PRECISION_EIGHTHS_FRACTIONAL, "1-3/8 in"),
        (
            "12.5 deg",
            "deg",
            PRECISION_SECONDS_TWO_DECIMAL_ANGLE,
            "12° 30' .00\"",
        ),
    ] {
        let mut parameters = exposed();
        parameters.set_expression(1, expression).unwrap();
        parameters
            .set_custom_property_format(
                1,
                CustomPropertyFormat {
                    units: units.into(),
                    precision,
                    show_leading_zeros: false,
                    show_trailing_zeros: true,
                    ..Default::default()
                },
            )
            .unwrap();
        let saved = package()
            .with_parameter_properties(
                &parameters,
                &DocumentProperties::default(),
                UnitsOfMeasure::default(),
            )
            .unwrap();
        let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
        let restored = reopened.restore_document_parameters().unwrap();
        let projected = reopened
            .restore_parameter_properties(&restored.parameters)
            .unwrap();
        assert_eq!(
            projected.get(USER_DEFINED, "width").unwrap().value,
            VariantValue::String(expected.into())
        );
        assert_eq!(
            projected.get(USER_DEFINED, "width").unwrap().from_parameter,
            "width"
        );
        assert_eq!(
            restored.parameters.get(1).unwrap().custom_property,
            parameters.get(1).unwrap().custom_property
        );
    }
}
