// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{CustomPropertyFormat, vocabulary::PRECISION_NAMES},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn decimal_properties_obey_units_and_zero_flags_without_changing_inputs() {
    let units = UnitsOfMeasure::default();
    let quantity = Quantity::new(-0.05, Dimension::Length);
    let mut format = CustomPropertyFormat::default();
    assert_eq!(
        format.format_numeric(quantity, &units).unwrap(),
        VariantValue::String("-0.5 mm".into())
    );
    format.show_trailing_zeros = true;
    format.show_leading_zeros = false;
    format.units = "cm".into();
    assert_eq!(
        format.format_numeric(quantity, &units).unwrap(),
        VariantValue::String("-.050 cm".into())
    );
    format.show_units_string = false;
    assert_eq!(
        format
            .format_numeric(Quantity::new(0.05, Dimension::Length), &units)
            .unwrap(),
        VariantValue::String(".050".into())
    );
    assert_eq!(quantity.value, -0.05);
    assert_eq!(units, UnitsOfMeasure::default());
}

#[test]
fn all_decimal_precisions_preserve_selected_number_values() {
    for precision in PRECISION_ZERO_DECIMAL_PLACE.0..=PRECISION_EIGHT_DECIMAL_PLACES.0 {
        let mut format = CustomPropertyFormat {
            precision: CustomPropertyPrecision(precision),
            show_trailing_zeros: true,
            ..Default::default()
        };
        let places = (precision - PRECISION_ZERO_DECIMAL_PLACE.0) as usize;
        let quantity = Quantity::new(1.25, Dimension::Unitless);
        assert_eq!(
            format
                .format_numeric(quantity, &UnitsOfMeasure::default())
                .unwrap(),
            VariantValue::String(format!("{:.places$}", 1.25))
        );
        format.property_type = CUSTOM_PROPERTY_NUMBER;
        assert_eq!(
            format
                .format_numeric(quantity, &UnitsOfMeasure::default())
                .unwrap(),
            VariantValue::Double(1.25)
        );
    }
}

#[test]
fn all_fractional_precisions_reduce_and_carry() {
    let units = UnitsOfMeasure::default();
    let expected = [
        "1", "1-1/2", "1-1/2", "1-3/8", "1-3/8", "1-3/8", "1-3/8", "1-3/8",
    ];
    for (index, expected) in expected.into_iter().enumerate() {
        let format = CustomPropertyFormat {
            units: "cm".into(),
            precision: CustomPropertyPrecision(PRECISION_ZERO_FRACTIONAL.0 + index as i32),
            ..Default::default()
        };
        assert_eq!(
            format
                .format_numeric(Quantity::new(1.375, Dimension::Length), &units)
                .unwrap(),
            VariantValue::String(format!("{expected} cm"))
        );
    }
    let format = CustomPropertyFormat {
        units: "cm".into(),
        precision: PRECISION_EIGHTHS_FRACTIONAL,
        ..Default::default()
    };
    assert_eq!(
        format
            .format_numeric(Quantity::new(-1.99, Dimension::Length), &units)
            .unwrap(),
        VariantValue::String("-2 cm".into())
    );
}

#[test]
fn all_angular_precisions_use_degrees_and_carry_at_rounding_boundaries() {
    let units = UnitsOfMeasure::default();
    let expected = [
        "13°",
        "12° 30'",
        "12° 30' 0\"",
        "12° 30' 0.0\"",
        "12° 30' 0.00\"",
        "12° 30' 0.000\"",
        "12° 30' 0.0000\"",
    ];
    for (index, expected) in expected.into_iter().enumerate() {
        let format = CustomPropertyFormat {
            units: "rad".into(),
            precision: CustomPropertyPrecision(PRECISION_DEGREES_ANGLE.0 + index as i32),
            show_trailing_zeros: true,
            ..Default::default()
        };
        assert_eq!(
            format
                .format_numeric(
                    Quantity::new(12.5f64.to_radians(), Dimension::Angle),
                    &units
                )
                .unwrap(),
            VariantValue::String(expected.into())
        );
    }
    let mut format = CustomPropertyFormat {
        precision: PRECISION_SECONDS_ANGLE,
        ..Default::default()
    };
    assert_eq!(
        format
            .format_numeric(
                Quantity::new(-12.99999f64.to_radians(), Dimension::Angle),
                &units
            )
            .unwrap(),
        VariantValue::String("-13° 0' 0\"".into())
    );
    format.show_units_string = false;
    assert_eq!(
        format
            .format_numeric(
                Quantity::new(12.5f64.to_radians(), Dimension::Angle),
                &units
            )
            .unwrap(),
        VariantValue::String("12 30 0".into())
    );
}

#[test]
fn format_checks_dimension_tags_units_and_nonfinite_or_oversized_values() {
    let units = UnitsOfMeasure::default();
    for format in [
        CustomPropertyFormat {
            property_type: CustomPropertyType(-1),
            ..Default::default()
        },
        CustomPropertyFormat {
            precision: CustomPropertyPrecision(0),
            ..Default::default()
        },
        CustomPropertyFormat {
            precision: PRECISION_SECONDS_ANGLE,
            ..Default::default()
        },
        CustomPropertyFormat {
            units: "missing".into(),
            ..Default::default()
        },
        CustomPropertyFormat {
            units: "rad".into(),
            ..Default::default()
        },
    ] {
        assert!(
            format
                .format_numeric(Quantity::new(1.0, Dimension::Length), &units)
                .is_err()
        );
    }
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY, f64::MAX] {
        assert!(
            CustomPropertyFormat::default()
                .format_numeric(Quantity::new(value, Dimension::Length), &units)
                .is_err()
        );
    }
    let fractional = CustomPropertyFormat {
        precision: PRECISION_HALF_FRACTIONAL,
        ..Default::default()
    };
    assert!(
        fractional
            .format_numeric(Quantity::new(1.0, Dimension::Angle), &units)
            .is_err()
    );
    assert!(
        fractional
            .format_numeric(Quantity::new(1e20, Dimension::Length), &units)
            .is_err()
    );
    assert!(
        CustomPropertyFormat::default()
            .format_numeric(Quantity::new(1.0, Dimension::Text), &units)
            .is_err()
    );
    assert_eq!(PRECISION_NAMES.len(), 24);
}

#[test]
fn working_scale_powers_and_negative_zero_are_preserved() {
    let units = UnitsOfMeasure::default().with_working_scale(0.01).unwrap();
    let format = CustomPropertyFormat {
        property_type: CUSTOM_PROPERTY_NUMBER,
        ..Default::default()
    };
    for (dimension, value) in [
        (Dimension::Length, 0.1),
        (Dimension::Area, 0.01),
        (Dimension::Volume, 0.001),
    ] {
        let VariantValue::Double(actual) = format
            .format_numeric(Quantity::new(1.0, dimension), &units)
            .unwrap()
        else {
            panic!("expected a number");
        };
        // Decimal unit factors are rounded binary64 values before their scale powers compose.
        assert!((actual - value).abs() <= 2.0 * f64::EPSILON * value);
    }
    let VariantValue::Double(value) = format
        .format_numeric(Quantity::new(-0.0, Dimension::Length), &units)
        .unwrap()
    else {
        panic!("expected a number");
    };
    assert_eq!(value.to_bits(), (-0.0f64).to_bits());
}
