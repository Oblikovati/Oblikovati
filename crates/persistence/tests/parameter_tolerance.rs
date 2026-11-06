// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::units::UnitsOfMeasure;
use oblikovati_persistence::package::{DocumentPackage, PackageCodecError, RestoredParameters};

fn restore(records: &str) -> Result<RestoredParameters, PackageCodecError> {
    DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: {records}\n").as_bytes(),
    )?
    .restore_parameters(UnitsOfMeasure::default())
}

#[test]
fn saved_fit_bands_survive_forward_references_and_nominal_edits() {
    let mut restored = restore("[{name: width, kind: user, expression: 'driver * 2', tolerance: {type: limitsFitsStacked, upper: 0.125, lower: -0.25, holeTolerance: H7, shaftTolerance: g6}, modelValueType: upper}, {name: driver, kind: user, expression: '10 mm'}]").unwrap();
    let tolerance = restored.parameters.get(1).unwrap().tolerance.clone();
    assert_eq!(tolerance.type_, TOLERANCE_LIMITS_FITS_STACKED);
    assert_eq!((tolerance.upper, tolerance.lower), (0.125, -0.25));
    assert_eq!(
        (&*tolerance.hole_tolerance, &*tolerance.shaft_tolerance),
        ("H7", "g6")
    );
    assert_eq!(restored.parameters.model_value(1).unwrap().value, 2.125);
    assert!(restored.parameters.drain_changed().is_empty());
    restored.parameters.set_expression(2, "20 mm").unwrap();
    assert_eq!(restored.parameters.get(1).unwrap().tolerance, tolerance);
    assert_eq!(restored.parameters.model_value(1).unwrap().value, 4.125);
}

#[test]
fn each_go_tolerance_name_restores_the_canonical_api_type() {
    for (name, expected) in [
        ("default", TOLERANCE_DEFAULT),
        ("override", TOLERANCE_OVERRIDE),
        ("symmetric", TOLERANCE_SYMMETRIC),
        ("deviation", TOLERANCE_DEVIATION),
        ("limitsStacked", TOLERANCE_LIMITS_STACKED),
        ("limitLinear", TOLERANCE_LIMIT_LINEAR),
        ("max", TOLERANCE_MAX),
        ("min", TOLERANCE_MIN),
        ("limitsFitsStacked", TOLERANCE_LIMITS_FITS_STACKED),
        ("limitsFitsLinear", TOLERANCE_LIMITS_FITS_LINEAR),
        ("limitsFitsShowSize", TOLERANCE_LIMITS_FITS_SHOW_SIZE),
        (
            "limitsFitsShowTolerance",
            TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE,
        ),
        ("basic", TOLERANCE_BASIC),
        ("reference", TOLERANCE_REFERENCE),
    ] {
        let restored = restore(&format!("[{{name: p, kind: reference, value: 10, unit: length, tolerance: {{type: {name}, upper: 0.5, lower: -0.5}}}}]")).unwrap();
        assert_eq!(
            restored.parameters.get(1).unwrap().tolerance.type_,
            expected
        );
        assert_eq!(restored.parameters.get(1).unwrap().tolerance.upper, 0.5);
    }
}

#[test]
fn selections_use_saved_deviations_without_converting_absolute_limits() {
    for (selection, expected) in [
        ("nominal", 10.0),
        ("lower", 9.75),
        ("upper", 10.5),
        ("median", 10.125),
    ] {
        let restored = restore(&format!("[{{name: p, kind: derived, value: 10, unit: length, tolerance: {{type: limitsStacked, upper: 0.5, lower: -0.25}}, modelValueType: {selection}}}]")).unwrap();
        assert_eq!(restored.parameters.model_value(1).unwrap().value, expected);
        assert_eq!(
            restored
                .parameters
                .get(1)
                .unwrap()
                .current_value()
                .unwrap()
                .value,
            10.0
        );
    }
}

#[test]
fn malformed_tolerances_and_non_numeric_selections_refuse_restoration() {
    for fields in [
        "tolerance: {}",
        "tolerance: {type: future}",
        "tolerance: {type: deviation, upper: -1, lower: 1}",
        "tolerance: {type: deviation, upper: .inf}",
        "tolerance: {type: deviation, lower: .nan}",
        "tolerance: {type: default, future: true}",
        "modelValueType: future",
    ] {
        assert!(
            restore(&format!(
                "[{{name: p, kind: user, expression: '1', {fields}}}]"
            ))
            .is_err(),
            "{fields}"
        );
    }
    for fields in ["tolerance: {type: default}", "modelValueType: nominal"] {
        assert!(
            restore(&format!(
                "[{{name: p, kind: user, valueType: text, text: example, {fields}}}]"
            ))
            .is_err()
        );
    }
    assert!(restore("[{name: p, kind: user, expression: '1e308', tolerance: {type: deviation, upper: 1e308}, modelValueType: upper}]").is_err());
}

#[test]
fn absent_and_null_tolerances_keep_defaults() {
    for suffix in ["", ", tolerance: null"] {
        let restored = restore(&format!(
            "[{{name: p, kind: user, expression: '1'{suffix}}}]"
        ))
        .unwrap();
        assert_eq!(
            restored.parameters.get(1).unwrap().tolerance,
            Tolerance::default()
        );
        assert_eq!(
            restored.parameters.get(1).unwrap().model_value_type,
            MODEL_VALUE_NOMINAL
        );
    }
}
