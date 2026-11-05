// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{DerivedTableHealth, ParameterCollection},
    property::{DocumentProperties, USER_DEFINED},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;
use std::collections::BTreeMap;

fn empty() -> DocumentPackage {
    DocumentPackage::empty(DOCUMENT_PART, "Part")
}

#[test]
fn failed_expression_and_blocked_reader_keep_last_good_values_tolerance_and_recover() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(7, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(9, "twice", USER_PARAM, "width * 2")
        .unwrap();
    parameters.set_deviation(7, 0.2, -0.1).unwrap();
    parameters
        .set_model_value_type(7, MODEL_VALUE_UPPER)
        .unwrap();
    let mut annotation = parameters.get(7).unwrap().annotations.clone();
    annotation.exposed_as_property = true;
    parameters.set_annotations(7, annotation).unwrap();
    parameters.set_expression(7, "missing + 1 cm").unwrap();
    let package = empty()
        .with_parameter_properties(
            &parameters,
            &DocumentProperties::default(),
            UnitsOfMeasure::default(),
        )
        .unwrap();
    let mut restored = DocumentPackage::decode(&package.encode().unwrap())
        .unwrap()
        .restore_document_parameters()
        .unwrap();
    assert_eq!(restored.parameters.get(7), parameters.get(7));
    assert_eq!(restored.parameters.get(9), parameters.get(9));
    assert!(
        package
            .restore_parameter_properties(&restored.parameters)
            .unwrap()
            .get(USER_DEFINED, "width")
            .is_none()
    );
    assert_eq!(
        restored
            .parameters
            .inspection_value(7)
            .unwrap()
            .unwrap()
            .value,
        4.2
    );
    let id = restored.ids.allocate().unwrap();
    restored
        .parameters
        .insert_expression(id, "missing", USER_PARAM, "5 cm")
        .unwrap();
    assert_eq!(restored.parameters.model_value(7).unwrap().value, 6.2);
    assert_eq!(
        restored
            .parameters
            .get(9)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        12.0
    );
}

#[test]
fn unavailable_and_missing_derived_links_keep_failure_until_source_refresh() {
    for missing in [false, true] {
        let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
        let values = BTreeMap::from([
            ("width".into(), Quantity::new(4.0, Dimension::Length)),
            ("height".into(), Quantity::new(2.0, Dimension::Length)),
        ]);
        parameters
            .add_derived_table(
                7,
                "source.opd",
                &["width".into(), "height".into()],
                &values,
                &BTreeMap::from([("width".into(), 1), ("height".into(), 2)]),
            )
            .unwrap();
        parameters
            .insert_expression(3, "area", USER_PARAM, "width * height")
            .unwrap();
        let partial = BTreeMap::from([("height".into(), values["height"])]);
        parameters
            .sync_derived_table(7, missing.then_some(&partial))
            .unwrap();
        let package = empty()
            .with_parameters(&parameters, UnitsOfMeasure::default())
            .unwrap();
        let mut restored = package.restore_document_parameters().unwrap().parameters;
        assert_eq!(restored.get(1), parameters.get(1));
        assert_eq!(restored.get(3), parameters.get(3));
        assert_eq!(
            restored.derived_table(7).unwrap().health,
            parameters.derived_table(7).unwrap().health
        );
        restored.sync_derived_table(7, Some(&values)).unwrap();
        assert_eq!(
            restored.derived_table(7).unwrap().health,
            DerivedTableHealth::Healthy
        );
        assert_eq!(restored.get(3).unwrap().current_value().unwrap().value, 8.0);
    }
}

#[test]
fn malformed_diagnostics_and_link_failure_metadata_are_refused() {
    for fields in [
        "expression: '2 cm', lastGood: {value: 9, unit: length}",
        "expression: missing, lastGood: {value: .nan, unit: length}",
        "expression: missing, lastGood: {value: 2, unit: text}",
        "expression: '(' ",
    ] {
        let package = DocumentPackage::decode(format!("schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: [{{name: width, kind: user, {fields}}}]\n").as_bytes()).unwrap();
        assert!(package.restore_document_parameters().is_err(), "{fields}");
    }
    for parameters in ["[]", "[absent]", "[width, width]"] {
        let package = DocumentPackage::decode(format!("schemaVersion: 3\ndocumentType: 1\nmodel:\n  derivedParameterTables: [{{id: 7, sourceDocument: source.opd, linked: [width], failure: {{state: missing, parameters: {parameters}}}}}]\n").as_bytes()).unwrap();
        assert!(package.restore_document_parameters().is_err());
    }
}

#[test]
fn evaluation_failures_and_diagnostic_signed_zero_round_trip_but_cycles_do_not() {
    for expression in ["missing", "1 / 0", "1e308 * 1e308"] {
        let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
        parameters
            .insert_value(
                1,
                "value",
                USER_PARAM,
                Quantity::new(-0.0, Dimension::Unitless),
            )
            .unwrap();
        parameters.set_expression(1, expression).unwrap();
        assert!(parameters.get(1).unwrap().current_value().is_none());
        let saved = empty()
            .with_parameters(&parameters, UnitsOfMeasure::default())
            .unwrap();
        let restored = saved.restore_document_parameters().unwrap();
        assert!(
            restored
                .parameters
                .get(1)
                .unwrap()
                .current_value()
                .is_none()
        );
        assert_eq!(
            restored
                .parameters
                .inspection_value(1)
                .unwrap()
                .unwrap()
                .value
                .to_bits(),
            (-0.0_f64).to_bits()
        );
    }
    let cyclic = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: [{name: a, kind: user, expression: b}, {name: b, kind: user, expression: a}]\n").unwrap();
    assert!(cyclic.restore_document_parameters().is_err());
}
