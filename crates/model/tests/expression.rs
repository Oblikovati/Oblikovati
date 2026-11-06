// SPDX-License-Identifier: GPL-2.0-only

use Dimension::*;
use oblikovati_model::{
    expression::{Expression, ExpressionError},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn compile(source: &str) -> Expression {
    Expression::parse(source, &UnitsOfMeasure::default()).unwrap()
}

#[test]
fn precedence_associativity_negation_and_units_match_go_cases() {
    let cases = [
        ("2 + 3 * 4", 14.0, Unitless),
        ("(2 + 3) * 4", 20.0, Unitless),
        ("-3 + 1", -2.0, Unitless),
        ("8/2/2", 2.0, Unitless),
        ("8-2-3", 3.0, Unitless),
        ("--3", 3.0, Unitless),
        ("2*-3", -6.0, Unitless),
        ("5 mm + 1 cm", 1.5, Length),
        ("2 cm * 3 cm", 6.0, Area),
        ("2e-1m", 20.0, Length),
    ];
    for (source, value, dimension) in cases {
        assert_eq!(
            compile(source).evaluate().unwrap(),
            Quantity::new(value, dimension),
            "{source}"
        );
    }
}

#[test]
fn syntax_errors_report_byte_position_and_expected_shape() {
    for (source, position) in [
        ("2 +", 3),
        ("2 @ 3", 2),
        ("(2 + 3", 6),
        ("sin(", 4),
        ("* 5", 0),
        ("+5", 0),
        ("1..2", 0),
        ("1e999", 0),
        ("1 cm^4", 2),
        ("max(1,)", 6),
        ("2 3", 2),
        ("width\n", 5),
        ("µm @ 2", 4),
    ] {
        let error = Expression::parse(source, &UnitsOfMeasure::default()).unwrap_err();
        assert!(
            matches!(error, ExpressionError::Syntax { position: actual, .. } if actual == position),
            "{source}: {error}"
        );
        assert!(error.to_string().contains("expected"));
    }
}

#[test]
fn constant_errors_are_deferred_to_evaluation() {
    for source in ["1 mm + 1 deg", "1/0", "unknown(3)", "sin()", "sqrt(3 cm)"] {
        let error = compile(source).evaluate().unwrap_err();
        assert!(error.to_string().contains("expected"), "{error}");
    }
    assert!(compile("sqrt(-1)").evaluate().unwrap().value.is_nan());
}

#[test]
fn folded_and_dynamic_paths_produce_the_same_bits() {
    let constant = compile("2 * (3 + 4) - 1").evaluate().unwrap();
    let formula = compile("factor * (3 + 4) - 1").bind(|_| Some(1_u64));
    let dynamic = formula
        .evaluate(|_| Some(Quantity::new(2.0, Unitless)))
        .unwrap();
    assert_eq!(dynamic.value.to_bits(), constant.value.to_bits());
    assert_eq!(dynamic.dimension, constant.dimension);
}

#[test]
fn long_flat_formula_evaluates_and_drops_without_recursion() {
    let source = std::iter::repeat_n("x", 20_000)
        .collect::<Vec<_>>()
        .join("+");
    let expression = compile(&source);
    assert_eq!(expression.references(), ["x"]);
    let bound = expression.bind(|_| Some(7_u64));
    assert_eq!(
        bound
            .evaluate(|_| Some(Quantity::new(1.0, Unitless)))
            .unwrap()
            .value,
        20_000.0
    );
}

#[test]
fn syntax_nesting_is_bounded_before_stack_exhaustion() {
    for source in [
        format!("{}1{}", "(".repeat(129), ")".repeat(129)),
        format!("{}1", "-".repeat(129)),
    ] {
        let error = Expression::parse(&source, &UnitsOfMeasure::default()).unwrap_err();
        assert!(error.to_string().contains("128 nested"));
    }
    assert!(
        Expression::parse(
            &format!("{}1{}", "(".repeat(128), ")".repeat(128)),
            &UnitsOfMeasure::default()
        )
        .is_ok()
    );
}

#[test]
fn scale_aware_literals_and_bound_values_share_working_units() {
    let units = UnitsOfMeasure::default().centered_on_length("µm").unwrap();
    let expression = Expression::parse("2 * width + 5 µm", &units)
        .unwrap()
        .bind(|_| Some(9_u64));
    let quantity = expression
        .evaluate(|_| Some(Quantity::new(10.0, Length)))
        .unwrap();
    assert_eq!(quantity, Quantity::new(25.0, Length));
    assert_eq!(
        Expression::parse("5", &units).unwrap().evaluate().unwrap(),
        Quantity::new(5.0, Unitless)
    );
}
