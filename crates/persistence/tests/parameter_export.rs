// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\ndata: {addin: YWJj}\nmodel: {features: [{id: 17, kind: retained}], extension: {value: future}}\n").unwrap()
}

#[test]
fn edited_values_and_all_parameter_kinds_round_trip_in_creation_order() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name, kind) in [
        (50, "u", USER_PARAM),
        (10, "m", MODEL_PARAM),
        (90, "t", TABLE_PARAM),
    ] {
        parameters
            .insert_expression(id, name, kind, "10 mm")
            .unwrap();
    }
    parameters
        .insert_value(
            80,
            "r",
            REFERENCE_PARAM,
            Quantity::new(2.0, Dimension::Area),
        )
        .unwrap();
    parameters
        .insert_value(
            70,
            "d",
            DERIVED_PARAM,
            Quantity::new(-0.0, Dimension::Volume),
        )
        .unwrap();
    parameters.insert_text(60, "label", "hello: world").unwrap();
    parameters.insert_boolean(40, "flag", true).unwrap();
    parameters.set_expression(50, "r / (2 cm)").unwrap();
    let before = package();
    let updated = before
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap();
    let restored = updated
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    for (before, after) in parameters.ids().iter().zip(restored.parameters.ids()) {
        assert_eq!(parameters.get(*before), restored.parameters.get(*after));
    }
    assert_eq!(
        restored
            .parameters
            .get(5)
            .unwrap()
            .current_value()
            .unwrap()
            .value
            .to_bits(),
        (-0.0_f64).to_bits()
    );
    assert_eq!(updated.streams(), before.streams());
    let model = String::from_utf8(updated.section_yaml("model").unwrap().unwrap()).unwrap();
    assert!(model.contains("kind: retained") && model.contains("value: future"));
    assert_eq!(before, package());
    assert_eq!(
        DocumentPackage::decode(&updated.encode().unwrap()).unwrap(),
        updated
    );
}

#[test]
fn rich_metadata_groups_settings_and_links_survive_export() {
    let yaml = "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters:\n    - {name: width, kind: user, expression: '20 mm', comment: note, key: true, export: true, precision: 7, hidden: true, renamed: true, disabledActions: [edit, rename, delete], displayFormat: architectural, expressionList: ['20 mm', '10 mm'], allowCustomValue: true, sortedValueList: true, tolerance: {type: limitsFitsLinear, upper: 0.125, lower: -0.25, holeTolerance: H7, shaftTolerance: g6}, modelValueType: median, customProperty: {propertyType: number, units: in, precision: eighthsFractional, showTrailingZeros: true}}\n    - {name: module, kind: derived, value: 0.2, unit: length}\n  parameterGroups: [{internalName: dimensions, displayName: Dimensions, clientId: addin, members: [width, module]}]\n  parameterSettings: {linearStandardTolerance: '+/-0.1', angularStandardTolerance: '+/-1deg', useStandardTolerances: true, exportStandardTolerances: true, linearDimensionPrecision: 5, angularDimensionPrecision: 1, dimensionDisplayType: tolerance, displayParameterAsExpression: true}\n  derivedParameterTables: [{id: 7, sourceDocument: gear.obk, linked: [module, absent], ownedByFeature: true}]\n";
    let original = DocumentPackage::decode(yaml.as_bytes()).unwrap();
    let original_parameters = original
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap()
        .parameters;
    let updated = original
        .with_parameters(&original_parameters, UnitsOfMeasure::default())
        .unwrap();
    let roundtrip = updated
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap()
        .parameters;
    assert_eq!(roundtrip.get(1), original_parameters.get(1));
    assert_eq!(roundtrip.groups(), original_parameters.groups());
    assert_eq!(roundtrip.settings(), original_parameters.settings());
    assert_eq!(
        roundtrip.derived_tables(),
        original_parameters.derived_tables()
    );
    assert_eq!(
        updated.encode().unwrap(),
        updated
            .with_parameters(&roundtrip, UnitsOfMeasure::default())
            .unwrap()
            .encode()
            .unwrap()
    );
}

#[test]
fn enum_spellings_round_trip_for_all_tolerances_selections_and_display_modes() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "10")
        .unwrap();
    for kind in TOLERANCE_DEFAULT.0..=TOLERANCE_REFERENCE.0 {
        parameters
            .set_tolerance(
                1,
                Tolerance {
                    type_: ToleranceType(kind),
                    upper: 0.2,
                    lower: -0.1,
                    ..Tolerance::default()
                },
            )
            .unwrap();
        for selection in [
            MODEL_VALUE_NOMINAL,
            MODEL_VALUE_UPPER,
            MODEL_VALUE_LOWER,
            MODEL_VALUE_MEDIAN,
        ] {
            parameters.set_model_value_type(1, selection).unwrap();
            assert!(
                package()
                    .with_parameters(&parameters, UnitsOfMeasure::default())
                    .is_ok()
            );
        }
    }
    for mode in [
        DIMENSION_DISPLAY_VALUE,
        DIMENSION_DISPLAY_NAME,
        DIMENSION_DISPLAY_EXPRESSION,
        DIMENSION_DISPLAY_TOLERANCE,
        DIMENSION_DISPLAY_PRECISE_VALUE,
    ] {
        let mut settings = parameters.settings().clone();
        settings.dimension_display_type = mode;
        parameters.set_settings(settings).unwrap();
        assert!(
            package()
                .with_parameters(&parameters, UnitsOfMeasure::default())
                .is_ok()
        );
    }
    for display in [
        DISPLAY_FORMAT_DECIMAL,
        DISPLAY_FORMAT_FRACTIONAL,
        DISPLAY_FORMAT_ARCHITECTURAL,
    ] {
        let mut annotations = parameters.get(1).unwrap().annotations.clone();
        annotations.display_format = display;
        parameters.set_annotations(1, annotations).unwrap();
        assert!(
            package()
                .with_parameters(&parameters, UnitsOfMeasure::default())
                .is_ok()
        );
    }
}

#[test]
fn each_custom_property_precision_round_trips() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "label", "value").unwrap();
    for precision in PRECISION_ZERO_DECIMAL_PLACE.0..=PRECISION_SECONDS_FOUR_DECIMAL_ANGLE.0 {
        let mut format = parameters.get(1).unwrap().custom_property.clone();
        format.precision = CustomPropertyPrecision(precision);
        parameters.set_custom_property_format(1, format).unwrap();
        assert!(
            package()
                .with_parameters(&parameters, UnitsOfMeasure::default())
                .is_ok()
        );
    }
}

#[test]
fn unrepresentable_state_refuses_export_and_failed_recipes_promote_without_mutation() {
    let original = package();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "10 mm")
        .unwrap();
    let other_scale = UnitsOfMeasure::default().with_working_scale(2.0).unwrap();
    assert!(original.with_parameters(&parameters, other_scale).is_err());
    parameters.set_expression(1, "missing").unwrap();
    let failed = original
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(failed.manifest().schema_version, 3);
    assert!(
        failed
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .is_none()
    );
    let mut measured = ParameterCollection::new(UnitsOfMeasure::default());
    measured
        .insert_value(
            1,
            "measured",
            USER_PARAM,
            Quantity::new(1.0, Dimension::Length),
        )
        .unwrap();
    assert!(
        original
            .with_parameters(&measured, UnitsOfMeasure::default())
            .is_ok()
    );
    let mut builtin = ParameterCollection::new(UnitsOfMeasure::default());
    builtin.insert_auto_model(1, "10 mm").unwrap();
    assert!(
        original
            .with_parameters(&builtin, UnitsOfMeasure::default())
            .is_err()
    );
    assert_eq!(original, package());
}

#[test]
fn renamed_derived_links_promote_the_recipe_and_preserve_source_mapping() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(
            1,
            "module",
            DERIVED_PARAM,
            Quantity::new(0.2, Dimension::Length),
        )
        .unwrap();
    parameters
        .restore_derived_table(7, "source", &["module".into()], false)
        .unwrap();
    parameters.rename(1, "local_module").unwrap();
    let encoded = package()
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(encoded.manifest().schema_version, 3);
    let restored = DocumentPackage::decode(&encoded.encode().unwrap())
        .unwrap()
        .restore_document_parameters()
        .unwrap();
    assert_eq!(
        restored.parameters.derived_references(7).unwrap(),
        parameters.derived_references(7).unwrap()
    );
}

#[test]
fn empty_table_clears_previous_parameter_sections() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "label", "old").unwrap();
    let populated = package()
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap();
    let cleared = populated
        .with_parameters(
            &ParameterCollection::new(UnitsOfMeasure::default()),
            UnitsOfMeasure::default(),
        )
        .unwrap();
    assert!(
        cleared
            .restore_parameters(UnitsOfMeasure::default())
            .unwrap()
            .parameters
            .ids()
            .is_empty()
    );
}

#[test]
fn fixture_accepted_by_go_keeps_the_same_recipe_on_rust_export() {
    let model = include_str!("fixtures/rust-parameter-export.yaml");
    let indented = model
        .lines()
        .map(|line| format!("  {line}\n"))
        .collect::<String>();
    let encoded = format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n{indented}");
    let original = DocumentPackage::decode(encoded.as_bytes()).unwrap();
    let restored = original
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    let updated = original
        .with_parameters(&restored.parameters, UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(
        updated
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1),
        restored.parameters.get(1)
    );
    assert_eq!(restored.parameters.model_value(1).unwrap().value, 2.6);
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        5.0
    );
}
