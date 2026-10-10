// SPDX-License-Identifier: GPL-2.0-only

use Dimension::*;
use oblikovati_model::quantity::{Dimension, Quantity, QuantityError};

const CATEGORIES: [Dimension; 9] = [
    Unitless, Length, Angle, Area, Volume, Mass, Time, Boolean, Text,
];
// Independent table of every representable product, rather than exponent logic.
const PRODUCTS: [(Dimension, Dimension, Dimension); 5] = [
    (Length, Length, Area),
    (Length, Area, Volume),
    (Area, Length, Volume),
    (Unitless, Unitless, Unitless),
    (Area, Unitless, Area),
];

fn product_dimension(left: Dimension, right: Dimension) -> Option<Dimension> {
    if matches!(left, Boolean | Text) || matches!(right, Boolean | Text) {
        return None;
    }
    if left == Unitless {
        return Some(right);
    }
    if right == Unitless {
        return Some(left);
    }
    PRODUCTS
        .iter()
        .find(|(a, b, _)| *a == left && *b == right)
        .map(|(_, _, result)| *result)
}

#[test]
fn all_category_pairs_preserve_addition_and_subtraction_rules() {
    for left in CATEGORIES {
        for right in CATEGORIES {
            let (a, b) = (Quantity::new(6.0, left), Quantity::new(2.0, right));
            assert_eq!(
                a.checked_add(b).ok(),
                (left == right).then_some(Quantity::new(8.0, left))
            );
            assert_eq!(
                a.checked_sub(b).ok(),
                (left == right).then_some(Quantity::new(4.0, left))
            );
        }
    }
}

#[test]
fn all_category_pairs_match_named_product_table() {
    for left in CATEGORIES {
        for right in CATEGORIES {
            let expected = product_dimension(left, right).map(|unit| Quantity::new(12.0, unit));
            assert_eq!(
                Quantity::new(6.0, left)
                    .checked_mul(Quantity::new(2.0, right))
                    .ok(),
                expected
            );
        }
    }
}

#[test]
fn all_category_pairs_match_inverse_product_table() {
    for numerator in CATEGORIES {
        for divisor in CATEGORIES {
            let expected = CATEGORIES
                .into_iter()
                .find(|candidate| product_dimension(*candidate, divisor) == Some(numerator));
            let actual = Quantity::new(6.0, numerator).checked_div(Quantity::new(2.0, divisor));
            assert_eq!(actual.ok(), expected.map(|unit| Quantity::new(3.0, unit)));
        }
    }
}

#[test]
fn category_arithmetic_and_negation_preserve_values() {
    for dimension in CATEGORIES {
        assert_eq!(
            dimension.is_arithmetic(),
            !matches!(dimension, Boolean | Text)
        );
        assert_eq!(
            Quantity::new(3.0, dimension).negated(),
            Quantity::new(-3.0, dimension)
        );
        assert_eq!(
            Quantity::new(0.0, dimension).negated().value.to_bits(),
            (-0.0_f64).to_bits()
        );
    }
}

#[test]
fn zero_division_precedes_dimension_validation_and_names_operands() {
    for zero in [0.0, -0.0] {
        let dividend = Quantity::new(12.0, Volume);
        let divisor = Quantity::new(zero, Text);
        let error = dividend.checked_div(divisor).unwrap_err();
        assert_eq!(error, QuantityError::ZeroDivisor { dividend, divisor });
        let message = error.to_string();
        assert!(
            message.contains("12.0") && message.contains("Text") && message.contains("nonzero")
        );
    }
}

#[test]
fn invalid_dimension_error_names_operator_and_categories() {
    let error = Quantity::new(2.0, Volume)
        .checked_mul(Quantity::new(3.0, Mass))
        .unwrap_err();
    assert_eq!(
        error,
        QuantityError::Incompatible {
            operator: "*",
            left: Volume,
            right: Mass
        }
    );
    let message = error.to_string();
    assert!(message.contains('*') && message.contains("Volume") && message.contains("Mass"));
    assert!(message.contains("expected") && message.contains("named result"));
}

#[test]
fn ieee_results_are_not_silently_clamped_or_rounded() {
    let huge = Quantity::new(f64::MAX, Length);
    assert!(huge.checked_add(huge).unwrap().value.is_infinite());
    let nan = Quantity::new(f64::NAN, Length);
    assert!(nan.checked_sub(huge).unwrap().value.is_nan());
    let fraction = Quantity::new(1.0, Volume)
        .checked_div(Quantity::new(3.0, Area))
        .unwrap();
    assert_eq!(fraction.value.to_bits(), (1.0_f64 / 3.0).to_bits());
}
