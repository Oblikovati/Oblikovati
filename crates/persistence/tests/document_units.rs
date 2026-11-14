// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::Dimension,
    units::{AngleFormat, UnitsOfMeasure},
};
use oblikovati_persistence::package::DocumentPackage;

fn package(units: &str) -> DocumentPackage {
    DocumentPackage::decode(format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  units: {units}\n  parameters: [{{name: width, kind: user, expression: '1 mm'}}, {{name: measured, kind: user, expression: '7 length'}}]\n").as_bytes()).unwrap()
}

#[test]
fn stored_context_controls_expression_values_without_rescaling_measurements() {
    let original = package(
        "{workingScale: '0.0001', length: in, area: 'cm^2', volume: 'cm^3', angle: rad, mass: g, time: min, lengthPrecision: '5', anglePrecision: '1', lengthFormat: architectural, angleFormat: dms}",
    );
    let units = original.document_units().unwrap();
    assert_eq!(units.working_scale(), 0.0001);
    assert_eq!(units.preferred_name(Dimension::Length), "in");
    assert_eq!(units.display_options().length_precision, 5);
    assert_eq!(units.display_options().angle_precision, 1);
    assert_eq!(
        units.display_options().length_format,
        DISPLAY_FORMAT_ARCHITECTURAL
    );
    assert_eq!(
        units.display_options().angle_format,
        AngleFormat::DegreesMinutesSeconds
    );
    let restored = original.restore_document_parameters().unwrap();
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1000.0
    );
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        7.0
    );
    let exported = original
        .with_parameters(&restored.parameters, units.clone())
        .unwrap();
    assert_eq!(exported.document_units().unwrap(), units);
    assert_eq!(
        exported
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1),
        restored.parameters.get(1)
    );
}

#[test]
fn absent_sparse_and_null_settings_retain_legacy_centimetre_defaults() {
    for settings in ["null", "{}", "{length: mm}"] {
        let original = package(settings);
        assert_eq!(
            original.document_units().unwrap(),
            UnitsOfMeasure::default()
        );
        assert_eq!(
            original
                .restore_document_parameters()
                .unwrap()
                .parameters
                .get(1)
                .unwrap()
                .current_value()
                .unwrap()
                .value,
            0.1
        );
    }
    let absent = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert_eq!(absent.document_units().unwrap(), UnitsOfMeasure::default());
}

#[test]
fn invalid_fields_are_refused_instead_of_replaced_during_export() {
    for settings in [
        "[]",
        "{future: mm}",
        "{unitless: mm}",
        "{length: deg}",
        "{lengthPrecision: '-1'}",
        "{anglePrecision: '9223372036854775808'}",
        "{lengthPrecision: bad}",
        "{lengthFormat: bad}",
        "{angleFormat: bad}",
        "{workingScale: nope}",
        "{workingScale: '0'}",
        "{workingScale: '-1'}",
        "{workingScale: NaN}",
        "{workingScale: '1e300'}",
        "{workingScale: '1e-300'}",
    ] {
        let original = package(settings);
        assert!(
            original.restore_document_parameters().is_err(),
            "{settings}"
        );
        assert!(
            original
                .with_parameters(
                    &ParameterCollection::new(UnitsOfMeasure::default()),
                    UnitsOfMeasure::default()
                )
                .is_err(),
            "{settings}"
        );
    }
}

#[test]
fn formats_and_supported_precision_boundaries_round_trip() {
    for length in ["decimal", "fractional", "architectural"] {
        for angle in ["decimal", "dms"] {
            let original = package(&format!(
                "{{lengthFormat: {length}, angleFormat: {angle}, lengthPrecision: '0', anglePrecision: '255'}}"
            ));
            let units = original.document_units().unwrap();
            let parameters = original.restore_document_parameters().unwrap().parameters;
            assert_eq!(
                original
                    .with_parameters(&parameters, units.clone())
                    .unwrap()
                    .document_units()
                    .unwrap(),
                units
            );
        }
    }
}

#[test]
fn new_non_default_scale_is_persisted_and_existing_model_scale_cannot_change() {
    let empty = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let units = UnitsOfMeasure::default()
        .with_working_scale(100000.0)
        .unwrap();
    let mut parameters = ParameterCollection::new(units.clone());
    parameters
        .insert_expression(1, "distance", USER_PARAM, "1 km")
        .unwrap();
    let exported = empty.with_parameters(&parameters, units.clone()).unwrap();
    let reopened = DocumentPackage::decode(&exported.encode().unwrap()).unwrap();
    assert_eq!(reopened.document_units().unwrap(), units);
    assert_eq!(
        reopened
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1.0
    );
    assert!(
        package("{}")
            .with_parameters(&parameters, units)
            .unwrap_err()
            .to_string()
            .contains("expected stored scale")
    );
}
