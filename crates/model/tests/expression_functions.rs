// SPDX-License-Identifier: GPL-2.0-only

use Dimension::*;
use oblikovati_model::{
    expression::Expression,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

const UNARY_NAMES: [&str; 14] = [
    "sin", "cos", "tan", "asin", "acos", "atan", "sqrt", "abs", "floor", "ceil", "round", "ln",
    "log", "exp",
];
const DIMENSIONS: [Dimension; 9] = [
    Unitless, Length, Angle, Area, Volume, Mass, Time, Boolean, Text,
];

fn evaluate(source: &str) -> Quantity {
    Expression::parse(source, &UnitsOfMeasure::default())
        .unwrap()
        .evaluate()
        .unwrap()
}

fn expected_unary_dimension(name: &str, input: Dimension) -> Option<Dimension> {
    match name {
        "sin" | "cos" | "tan" if matches!(input, Unitless | Angle) => Some(Unitless),
        "asin" | "acos" | "atan" if input == Unitless => Some(Angle),
        "ln" | "log" | "exp" if input == Unitless => Some(Unitless),
        "sqrt" if input == Area => Some(Length),
        "sqrt" if input == Unitless => Some(Unitless),
        "abs" | "floor" | "ceil" | "round" => Some(input),
        _ => None,
    }
}

#[test]
fn every_unary_function_checks_every_dimension() {
    for name in UNARY_NAMES {
        let expression = Expression::parse(&format!("{name}(x)"), &UnitsOfMeasure::default())
            .unwrap()
            .bind(|_| Some(1_u64));
        for dimension in DIMENSIONS {
            let result = expression.evaluate(|_| Some(Quantity::new(1.0, dimension)));
            assert_eq!(
                result.as_ref().ok().map(|q| q.dimension),
                expected_unary_dimension(name, dimension),
                "{name}: {dimension:?}"
            );
            if let Err(error) = result {
                assert!(error.to_string().contains("expected"));
            }
        }
    }
}

#[test]
fn all_function_arities_are_checked_at_evaluation() {
    for name in UNARY_NAMES.into_iter().chain(["atan2", "min", "max"]) {
        let source = format!("{name}()");
        let expression = Expression::parse(&source, &UnitsOfMeasure::default()).unwrap();
        let error = expression.evaluate().unwrap_err();
        assert!(error.to_string().contains(name) && error.to_string().contains('0'));
    }
    for source in ["sin(1,2)", "min(1)", "max(1)", "atan2(1)", "atan2(1,2,3)"] {
        assert!(
            Expression::parse(source, &UnitsOfMeasure::default())
                .unwrap()
                .evaluate()
                .is_err()
        );
    }
}

#[test]
fn variadic_functions_preserve_dimensions_and_reject_mixed_inputs() {
    assert_eq!(
        evaluate("min(3 mm, 1 mm, 2 mm)"),
        Quantity::new(0.1, Length)
    );
    assert_eq!(
        evaluate("max(3 mm, 1 mm, 5 mm)"),
        Quantity::new(0.5, Length)
    );
    assert_eq!(evaluate("sqrt(2 cm * 2 cm)"), Quantity::new(2.0, Length));
    assert_eq!(
        evaluate("atan2(1 cm, 1 cm)"),
        Quantity::new(std::f64::consts::FRAC_PI_4, Angle)
    );
    for source in ["min(1 cm, 1 deg)", "max(1 cm, 1 deg)", "atan2(1 cm, 1 deg)"] {
        assert!(
            Expression::parse(source, &UnitsOfMeasure::default())
                .unwrap()
                .evaluate()
                .is_err()
        );
    }
}

#[test]
fn extrema_preserve_go_nan_infinity_and_signed_zero_rules() {
    for source in ["min(1, sqrt(-1))", "max(sqrt(-1), 1)"] {
        assert!(evaluate(source).value.is_nan());
    }
    assert_eq!(
        evaluate("min(-exp(1000), sqrt(-1))").value,
        f64::NEG_INFINITY
    );
    assert_eq!(evaluate("max(sqrt(-1), exp(1000))").value, f64::INFINITY);
    for (source, expected) in [
        ("min(0,-0)", -0.0_f64),
        ("min(-0,0)", -0.0),
        ("max(0,-0)", 0.0),
        ("max(-0,0)", 0.0),
        ("min(0,0)", 0.0),
        ("max(-0,-0)", -0.0),
    ] {
        assert_eq!(
            evaluate(source).value.to_bits(),
            expected.to_bits(),
            "{source}"
        );
    }
}
