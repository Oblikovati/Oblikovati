// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn large_precision_matches_go_formatting_for_normal_and_subnormal_values() {
    let mut units = UnitsOfMeasure::default();
    units.set_preferred(Dimension::Length, "cm").unwrap();
    let mut count = 0;
    for row in include_str!("fixtures/go-large-precision.tsv")
        .lines()
        .filter(|row| !row.starts_with('#'))
    {
        let fields: Vec<_> = row.split('\t').collect();
        let mut options = units.display_options();
        options.length_precision = fields[0].parse().unwrap();
        units.set_display_options(options).unwrap();
        let value = f64::from_bits(u64::from_str_radix(fields[1], 16).unwrap());
        assert_eq!(
            units
                .format_display(Quantity::new(value, Dimension::Length))
                .unwrap(),
            fields[2]
        );
        count += 1;
    }
    assert_eq!(count, 9);
}

#[test]
fn precision_validation_is_atomic_and_output_limits_do_not_change_preferences() {
    let mut units = UnitsOfMeasure::default();
    let mut options = units.display_options();
    options.length_precision = i64::MAX;
    options.angle_precision = i64::MAX;
    units.set_display_options(options).unwrap();
    for dimension in [Dimension::Length, Dimension::Angle] {
        let error = units
            .format_display(Quantity::new(1.0, dimension))
            .unwrap_err()
            .to_string();
        assert!(error.contains("9223372036854775807") && error.contains("1000000"));
    }
    assert_eq!(units.display_options(), options);
    options.angle_precision = -1;
    assert!(units.set_display_options(options).is_err());
    assert_eq!(units.display_options().angle_precision, i64::MAX);
}

#[test]
fn high_precision_rounding_does_not_overflow_power_of_ten() {
    let mut units = UnitsOfMeasure::default();
    units.set_preferred(Dimension::Length, "cm").unwrap();
    let mut options = units.display_options();
    options.length_precision = 1000;
    units.set_display_options(options).unwrap();
    assert_eq!(
        units
            .display_rounded_expression(Quantity::new(f64::from_bits(1), Dimension::Length))
            .unwrap(),
        "5e-324 cm"
    );
}
