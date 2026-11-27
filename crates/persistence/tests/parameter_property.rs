// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{parameter::CustomPropertyFormat, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

fn restore(
    record: &str,
) -> Result<CustomPropertyFormat, oblikovati_persistence::package::PackageCodecError> {
    let input = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{{name: size, kind: user, expression: '1', customProperty: {record}}}]\n"
    );
    let restored =
        DocumentPackage::decode(input.as_bytes())?.restore_parameters(UnitsOfMeasure::default())?;
    Ok(restored.parameters.get(1).unwrap().custom_property.clone())
}

#[test]
fn all_precision_spellings_match_the_go_contract_ids() {
    let rows: Vec<&str> = include_str!("fixtures/go-property-precision.tsv")
        .lines()
        .filter(|line| !line.starts_with('#'))
        .collect();
    assert_eq!(rows.len(), 24);
    for row in rows {
        let (name, number) = row.split_once('\t').unwrap();
        let format = restore(&format!("{{precision: {name}}}")).unwrap();
        assert_eq!(format.precision.0, number.parse::<i32>().unwrap());
    }
}

#[test]
fn absent_format_defaults_differ_from_explicit_empty_record_as_in_go() {
    assert_eq!(restore("null").unwrap(), CustomPropertyFormat::default());
    let explicit = restore("{}").unwrap();
    assert_eq!(explicit.property_type, CUSTOM_PROPERTY_TEXT);
    assert_eq!(explicit.precision, PRECISION_THREE_DECIMAL_PLACES);
    assert!(
        !explicit.show_leading_zeros
            && !explicit.show_trailing_zeros
            && !explicit.show_units_string
    );
}

#[test]
fn property_type_units_and_all_flags_are_restored() {
    let format = restore("{propertyType: number, units: mm, precision: eighthsFractional, showLeadingZeros: true, showTrailingZeros: true, showUnitsString: true}").unwrap();
    assert_eq!(format.property_type, CUSTOM_PROPERTY_NUMBER);
    assert_eq!(format.units, "mm");
    assert_eq!(format.precision, PRECISION_EIGHTHS_FRACTIONAL);
    assert!(format.show_leading_zeros && format.show_trailing_zeros && format.show_units_string);
    assert_eq!(
        restore("{propertyType: text}").unwrap().property_type,
        CUSTOM_PROPERTY_TEXT
    );
}

#[test]
fn malformed_format_records_are_refused_instead_of_dropped() {
    for record in [
        "{propertyType: unknown}",
        "{precision: unknown}",
        "{future: true}",
        "[]",
    ] {
        assert!(restore(record).is_err(), "{record}");
    }
    assert!(
        restore("{precision: unknown}")
            .unwrap_err()
            .to_string()
            .contains("unknown")
    );
}

#[test]
fn literal_and_read_only_parameters_accept_persisted_format_metadata() {
    for fields in [
        "kind: user, valueType: text, text: name",
        "kind: reference, value: 2, unit: length",
    ] {
        let input = format!(
            "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{{name: p, {fields}, customProperty: {{units: cm}}}}]\n"
        );
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        let restored = package
            .restore_parameters(UnitsOfMeasure::default())
            .unwrap();
        assert_eq!(
            restored.parameters.get(1).unwrap().custom_property.units,
            "cm"
        );
    }
}
