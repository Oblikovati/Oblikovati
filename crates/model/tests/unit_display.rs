// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    expression::Expression,
    quantity::{Dimension, Quantity},
    units::{AngleFormat, DisplayOptions, UnitsOfMeasure},
};

fn configured(mode: &str) -> (UnitsOfMeasure, Dimension) {
    let mut units = UnitsOfMeasure::default();
    let mut options = units.display_options();
    let mut dimension = Dimension::Length;
    match mode {
        "fraction" => {
            units.set_preferred(dimension, "in").unwrap();
            options.length_format = DISPLAY_FORMAT_FRACTIONAL;
            options.length_precision = 4;
        }
        "architectural" => {
            options.length_format = DISPLAY_FORMAT_ARCHITECTURAL;
            options.length_precision = 2;
        }
        "dms" => {
            options.angle_format = AngleFormat::DegreesMinutesSeconds;
            dimension = Dimension::Angle;
        }
        "decimal" => (),
        _ => panic!("unknown fixture mode {mode}"),
    }
    units.set_display_options(options).unwrap();
    (units, dimension)
}

#[test]
fn all_display_and_rounded_strings_match_the_fixed_go_capture() {
    let rows: Vec<&str> = include_str!("fixtures/go-display.tsv")
        .lines()
        .filter(|row| !row.starts_with('#'))
        .collect();
    assert_eq!(rows.len(), 48);
    for row in rows {
        let fields: Vec<&str> = row.split('\t').collect();
        let (units, dimension) = configured(fields[0]);
        let quantity = Quantity::new(
            f64::from_bits(u64::from_str_radix(fields[1], 16).unwrap()),
            dimension,
        );
        assert_eq!(units.format_display(quantity).unwrap(), fields[2], "{row}");
        assert_eq!(
            units.display_rounded_expression(quantity).unwrap(),
            fields[3],
            "{row}"
        );
        assert_eq!(units.format_precise(quantity).unwrap(), fields[4], "{row}");
    }
}

#[test]
fn architectural_display_and_rounded_source_honor_working_scale() {
    for scale in [0.001, 0.1, 1.0, 100.0] {
        let (units, _) = configured("architectural");
        let units = units.with_working_scale(scale).unwrap();
        let quantity = Quantity::new(34.29 / scale, Dimension::Length);
        assert_eq!(units.format_display(quantity).unwrap(), "1' 1-1/2\"");
        let source = units.display_rounded_expression(quantity).unwrap();
        assert_eq!(source, "13.5 in");
        assert_eq!(
            Expression::parse(&source, &units)
                .unwrap()
                .evaluate()
                .unwrap()
                .value,
            13.5 * 2.54 / scale
        );
    }
}

#[test]
fn fraction_clamps_carries_and_dms_carries_match_expected_units() {
    let (mut units, dimension) = configured("fraction");
    for precision in [0, 20, 255] {
        units
            .set_display_options(DisplayOptions {
                length_precision: precision,
                ..units.display_options()
            })
            .unwrap();
        assert_eq!(
            units
                .format_display(Quantity::new(1.5 * 2.54, dimension))
                .unwrap(),
            "1-1/2 in"
        );
    }
    let (units, _) = configured("architectural");
    assert_eq!(
        units
            .format_display(Quantity::new(11.99 * 2.54, dimension))
            .unwrap(),
        "1' 0\""
    );
    let (units, dimension) = configured("dms");
    for (degrees, expected) in [(29.99999, "30° 0' 0\""), (-30.5, "-30° 30' 0\"")] {
        assert_eq!(
            units
                .format_display(Quantity::new(
                    degrees * std::f64::consts::PI / 180.0,
                    dimension
                ))
                .unwrap(),
            expected
        );
    }
}

#[test]
fn precise_notation_and_fallback_categories_round_trip() {
    let units = UnitsOfMeasure::default();
    for (value, expected) in [
        (1e-5, "1e-05"),
        (1e-4, "0.0001"),
        (1e6, "1e+06"),
        (1e20, "1e+20"),
        (-0.0, "-0"),
    ] {
        let quantity = Quantity::new(value, Dimension::Unitless);
        assert_eq!(units.format_precise(quantity).unwrap(), expected);
        assert_eq!(
            Expression::parse(expected, &units)
                .unwrap()
                .evaluate()
                .unwrap()
                .value
                .to_bits(),
            value.to_bits()
        );
    }
    for dimension in [
        Dimension::Area,
        Dimension::Volume,
        Dimension::Mass,
        Dimension::Time,
    ] {
        let quantity = Quantity::new(2.0, dimension);
        assert_eq!(
            units.format_display(quantity).unwrap(),
            units.format_precise(quantity).unwrap()
        );
        assert_eq!(
            units.display_rounded_expression(quantity).unwrap(),
            units.format_precise(quantity).unwrap()
        );
    }
}

#[test]
fn invalid_options_and_nonfinite_or_excessive_values_are_named_errors() {
    let mut units = UnitsOfMeasure::default();
    let original = units.display_options();
    assert!(
        units
            .set_display_options(DisplayOptions {
                length_format: ParameterDisplayFormat(0),
                ..original
            })
            .is_err()
    );
    assert_eq!(units.display_options(), original);
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        assert!(
            units
                .format_display(Quantity::new(value, Dimension::Length))
                .is_err()
        );
    }
    for dimension in [Dimension::Text, Dimension::Boolean] {
        assert!(units.format_precise(Quantity::new(0.0, dimension)).is_err());
    }
    for mode in ["fraction", "architectural", "dms"] {
        let (units, dimension) = configured(mode);
        let error = units
            .format_display(Quantity::new(1e18, dimension))
            .unwrap_err()
            .to_string();
        assert!(error.contains("expected") && error.contains("tick count"));
    }
    assert!(
        units
            .display_rounded_expression(Quantity::new(f64::MAX, Dimension::Angle))
            .is_err()
    );
}

#[test]
fn preferences_are_clone_owned_and_decimal_precision_is_explicit() {
    let units = UnitsOfMeasure::default();
    let mut clone = units.clone();
    clone
        .set_display_options(DisplayOptions {
            length_precision: 0,
            angle_precision: 0,
            ..clone.display_options()
        })
        .unwrap();
    assert_eq!(units.display_options(), DisplayOptions::default());
    assert_eq!(
        clone
            .format_display(Quantity::new(1.23456, Dimension::Length))
            .unwrap(),
        "12 mm"
    );
    assert_eq!(
        units
            .format_display(Quantity::new(1.23456, Dimension::Length))
            .unwrap(),
        "12.346 mm"
    );
}

#[test]
fn decimal_angles_and_rounded_decimal_source_use_display_precision() {
    let units = UnitsOfMeasure::default();
    let angle = Quantity::new(45.0 * std::f64::consts::PI / 180.0, Dimension::Angle);
    assert_eq!(units.format_display(angle).unwrap(), "45.00 deg");
    assert_eq!(units.display_rounded_expression(angle).unwrap(), "45 deg");
    assert_eq!(
        units
            .display_rounded_expression(Quantity::new(0.9999999998, Dimension::Length))
            .unwrap(),
        "10 mm"
    );
    assert!(
        units
            .display_rounded_expression(Quantity::new(1e306, Dimension::Length))
            .is_err()
    );
}
