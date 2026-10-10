// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    expression::Expression,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn expressions_match_the_fixed_go_source_capture() {
    let corpus = include_str!("fixtures/go-expressions.tsv");
    let mut checked = 0;
    for row in corpus.lines().filter(|row| !row.starts_with('#')) {
        check_oracle_row(row);
        checked += 1;
    }
    assert_eq!(checked, 50);
}

fn check_oracle_row(row: &str) {
    let columns: Vec<&str> = row.split('\t').collect();
    let source = columns[3];
    let parsed = Expression::parse(source, &UnitsOfMeasure::default());
    if columns[0] == "parse" {
        assert!(parsed.is_err(), "{source}");
        return;
    }
    let result = parsed.unwrap().evaluate();
    if columns[0] == "eval" {
        assert!(result.is_err(), "{source}");
        return;
    }
    let quantity = result.unwrap();
    assert_eq!(quantity.dimension, oracle_dimension(columns[1]));
    let expected = f64::from_bits(u64::from_str_radix(columns[2], 16).unwrap());
    check_oracle_value(source, quantity, expected);
}

fn check_oracle_value(source: &str, quantity: Quantity, expected: f64) {
    if expected.is_nan() {
        assert!(quantity.value.is_nan(), "{source}");
        return;
    }
    let transcendental = [
        "sin(", "cos(", "tan(", "asin(", "acos(", "atan(", "ln(", "exp(",
    ];
    if expected.is_finite() && transcendental.iter().any(|name| source.starts_with(name)) {
        let relative_error = ((quantity.value - expected) / expected).abs();
        assert!(
            relative_error <= 8.0 * f64::EPSILON,
            "{source}: {} != {expected}",
            quantity.value
        );
        return;
    }
    assert_eq!(quantity.value.to_bits(), expected.to_bits(), "{source}");
}

fn oracle_dimension(name: &str) -> Dimension {
    match name {
        "unitless" => Dimension::Unitless,
        "length" => Dimension::Length,
        "area" => Dimension::Area,
        "volume" => Dimension::Volume,
        "angle" => Dimension::Angle,
        _ => panic!("unexpected fixture dimension {name:?}"),
    }
}
