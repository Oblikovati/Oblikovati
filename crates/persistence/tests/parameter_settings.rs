// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{parameter::CollectionSettings, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

fn package(settings: &str) -> DocumentPackage {
    let yaml = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{{name: width, kind: user, expression: '2'}}]\n  parameterSettings: {settings}\n"
    );
    DocumentPackage::decode(yaml.as_bytes()).unwrap()
}

#[test]
fn all_settings_fields_are_restored_without_numeric_recomputation() {
    let package = package(
        "{linearStandardTolerance: '+/-0.1', angularStandardTolerance: '+/-1deg', useStandardTolerances: true, exportStandardTolerances: true, linearDimensionPrecision: 8, angularDimensionPrecision: 0, dimensionDisplayType: expression, displayParameterAsExpression: true}",
    );
    let mut restored = package
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    let settings = restored.parameters.settings();
    assert_eq!(settings.linear_standard_tolerance, "+/-0.1");
    assert_eq!(settings.angular_standard_tolerance, "+/-1deg");
    assert!(settings.use_standard_tolerances && settings.export_standard_tolerances);
    assert_eq!(settings.linear_dimension_precision, 8);
    assert_eq!(settings.angular_dimension_precision, 0);
    assert_eq!(
        settings.dimension_display_type,
        DIMENSION_DISPLAY_EXPRESSION
    );
    assert!(settings.display_parameter_as_expression);
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert!(restored.parameters.drain_changed().is_empty());
}

#[test]
fn each_display_mode_maps_to_the_canonical_api_enum() {
    for (name, expected) in [
        ("value", DIMENSION_DISPLAY_VALUE),
        ("name", DIMENSION_DISPLAY_NAME),
        ("expression", DIMENSION_DISPLAY_EXPRESSION),
        ("tolerance", DIMENSION_DISPLAY_TOLERANCE),
        ("preciseValue", DIMENSION_DISPLAY_PRECISE_VALUE),
    ] {
        let package = package(&format!("{{dimensionDisplayType: {name}}}"));
        let restored = package
            .restore_parameters(UnitsOfMeasure::default())
            .unwrap();
        assert_eq!(
            restored.parameters.settings().dimension_display_type,
            expected
        );
        assert_eq!(restored.parameters.settings().linear_dimension_precision, 0);
        assert_eq!(
            restored.parameters.settings().angular_dimension_precision,
            0
        );
    }
}

#[test]
fn null_retains_defaults_but_present_empty_settings_are_invalid_like_go() {
    let restored = package("null")
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(
        restored.parameters.settings(),
        &CollectionSettings::default()
    );
    assert!(
        package("{}")
            .restore_parameters(UnitsOfMeasure::default())
            .is_err()
    );
}

#[test]
fn invalid_precision_mode_or_unknown_fields_refuse_reconstruction() {
    for record in [
        "{dimensionDisplayType: unknown}",
        "{dimensionDisplayType: value, linearDimensionPrecision: -1}",
        "{dimensionDisplayType: value, angularDimensionPrecision: 9}",
        "{dimensionDisplayType: value, future: true}",
        "[]",
    ] {
        assert!(
            package(record)
                .restore_parameters(UnitsOfMeasure::default())
                .is_err(),
            "{record}"
        );
    }
}
