// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_PARAM, REFERENCE_PARAM, TABLE_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::{UnitsOfMeasure, category_name},
};
use oblikovati_persistence::package::DocumentPackage;

const NUMERIC: [Dimension; 7] = [
    Dimension::Unitless,
    Dimension::Length,
    Dimension::Angle,
    Dimension::Area,
    Dimension::Volume,
    Dimension::Mass,
    Dimension::Time,
];

fn package(model: &str) -> DocumentPackage {
    let indented = model
        .lines()
        .map(|line| format!("  {line}\n"))
        .collect::<String>();
    DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n{indented}").as_bytes(),
    )
    .unwrap()
}

#[test]
fn go_measured_recipe_loads_all_categories_without_reinterpreting_units() {
    let package = package(include_str!("fixtures/go-measured-parameters.yaml"));
    let restored = package
        .restore_parameters(UnitsOfMeasure::default().with_working_scale(1e-4).unwrap())
        .unwrap();
    for (index, dimension) in NUMERIC.into_iter().enumerate() {
        assert_eq!(
            restored
                .parameters
                .get(index as u64 + 1)
                .unwrap()
                .current_value(),
            Some(Quantity::new(1.25, dimension))
        );
    }
}

#[test]
fn measured_values_round_trip_bit_for_bit_for_every_kind_scale_and_category() {
    let values = [
        0.0,
        -0.0,
        1.0 / 3.0,
        -123.456,
        f64::from_bits(1),
        f64::MIN_POSITIVE,
        f64::MAX,
    ];
    for scale in [1e-4, 1.0, 1e5] {
        let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
        let mut parameters = ParameterCollection::new(units.clone());
        let mut expected = Vec::new();
        for kind in [USER_PARAM, MODEL_PARAM, TABLE_PARAM] {
            for dimension in NUMERIC {
                for value in values {
                    let id = expected.len() as u64 + 1;
                    parameters
                        .insert_value(id, &format!("p{id}"), kind, Quantity::new(value, dimension))
                        .unwrap();
                    expected.push((value, dimension));
                }
            }
        }
        let exported = package("{}")
            .with_parameters(&parameters, units.clone())
            .unwrap();
        let reloaded = DocumentPackage::decode(&exported.encode().unwrap())
            .unwrap()
            .restore_parameters(units)
            .unwrap();
        for (index, (value, dimension)) in expected.into_iter().enumerate() {
            let actual = reloaded
                .parameters
                .get(index as u64 + 1)
                .unwrap()
                .current_value()
                .unwrap();
            assert_eq!(
                actual.value.to_bits(),
                value.to_bits(),
                "scale={scale}, dimension={dimension:?}"
            );
            assert_eq!(actual.dimension, dimension);
        }
    }
}

#[test]
fn copied_measurements_remain_editable_after_export_and_reload() {
    let units = UnitsOfMeasure::default()
        .with_working_scale(0.0001)
        .unwrap();
    let mut parameters = ParameterCollection::new(units.clone());
    parameters
        .insert_value(
            1,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(7.125, Dimension::Area),
        )
        .unwrap();
    parameters.copy_to_user(1, 2).unwrap();
    parameters
        .insert_expression(3, "twice", USER_PARAM, "measured_copy * 2")
        .unwrap();
    let exported = package("{}")
        .with_parameters(&parameters, units.clone())
        .unwrap();
    let mut reloaded = exported.restore_parameters(units).unwrap();
    assert_eq!(
        reloaded
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        7.125
    );
    reloaded.parameters.set_expression(2, "3 mm^2").unwrap();
    assert_eq!(
        reloaded
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        6_000_000.0
    );
}

#[test]
fn category_names_do_not_change_authored_expression_parsing_or_reference_binding() {
    let original = package(
        "parameters: [{name: length, kind: user, expression: '2'}, {name: result, kind: user, expression: 'length * 3'}, {name: literal, kind: user, expression: '2 mm'}]",
    );
    let restored = original
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        6.0
    );
    assert_eq!(
        restored
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        0.2
    );
    let mut parameters = restored.parameters;
    assert!(
        parameters
            .insert_expression(4, "diagnostic", USER_PARAM, "2 length")
            .is_err()
    );
}

#[test]
fn malformed_or_non_numeric_saved_constants_are_refused() {
    for source in [
        "NaN length",
        "inf mass",
        "1e999 area",
        "bad volume",
        "1 boolean",
        "2 text",
        "1 length extra",
        "1 unsupported",
    ] {
        let package = package(&format!(
            "parameters: [{{name: p, kind: user, expression: '{source}'}}]"
        ));
        assert!(
            package
                .restore_parameters(UnitsOfMeasure::default())
                .is_err(),
            "{source}"
        );
    }
}

#[test]
fn earlier_rust_diagnostic_capitalization_normalizes_to_stable_category_names() {
    for dimension in NUMERIC {
        let package = package(&format!(
            "parameters: [{{name: p, kind: user, expression: '1.25 {dimension:?}'}}]"
        ));
        let restored = package
            .restore_parameters(UnitsOfMeasure::default())
            .unwrap();
        assert_eq!(
            restored.parameters.get(1).unwrap().source,
            format!("1.25 {}", category_name(dimension))
        );
        assert!(
            package
                .with_parameters(&restored.parameters, UnitsOfMeasure::default())
                .is_ok()
        );
    }
}
