// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::ad::{DerivativeError, DualNumber};

fn close(actual: f64, expected: f64, tolerance: f64) {
    assert!(
        (actual - expected).abs() <= tolerance,
        "actual {actual:?}, expected {expected:?}, tolerance {tolerance}"
    );
}

fn evaluate(operation: &str, values: &[DualNumber]) -> Result<DualNumber, DerivativeError> {
    let (a, b) = (&values[0], &values[1]);
    match operation {
        "sum" => a.plus(b),
        "difference" => a.minus(b),
        "product" => a.product(b),
        "quotient" => a.quotient(b),
        "negation" => a.negated(),
        "offset" => a.shifted(3.0),
        "scale" => a.scaled(2.5),
        "sqrt" => a.sqrt(),
        "hypot" => a.hypot(b),
        "absolute" => a.abs(),
        "atan2" => a.atan2(b),
        "sine" => a.sin(),
        "cosine" => a.cos(),
        _ => panic!("unknown test operation {operation:?}"),
    }
}

fn independent_value(operation: &str, a: f64, b: f64) -> f64 {
    match operation {
        "sum" => a + b,
        "difference" => a - b,
        "product" => a * b,
        "quotient" => a / b,
        "negation" => -a,
        "offset" => a + 3.0,
        "scale" => a * 2.5,
        "sqrt" => a.sqrt(),
        "hypot" => a.hypot(b),
        "absolute" => a.abs(),
        "atan2" => a.atan2(b),
        "sine" => a.sin(),
        "cosine" => a.cos(),
        _ => panic!("unknown test operation {operation:?}"),
    }
}

#[test]
fn scalar_primitives_match_independent_central_differences_and_go_values() {
    let variables = DualNumber::seed(&[3.0, 4.0]).unwrap();
    for row in include_str!("fixtures/go-ad.tsv").lines() {
        let fields: Vec<&str> = row.split('\t').collect();
        let operation = fields[0];
        let result = evaluate(operation, &variables).unwrap();
        close(result.value(), fields[1].parse().unwrap(), 2e-15);
        for index in 0..2 {
            let expected: f64 = fields[index + 2].parse().unwrap();
            close(result.gradient()[index], expected, 2e-15);
            let mut plus = [3.0, 4.0];
            let mut minus = plus;
            let step = 1e-5;
            plus[index] += step;
            minus[index] -= step;
            let finite_difference = (independent_value(operation, plus[0], plus[1])
                - independent_value(operation, minus[0], minus[1]))
                / (2.0 * step);
            close(result.gradient()[index], finite_difference, 1e-9);
        }
    }
}

#[test]
fn seeds_constants_and_compositions_own_their_gradients() {
    let variables = DualNumber::seed(&[3.0, 4.0]).unwrap();
    assert_eq!(variables[0].gradient(), [1.0, 0.0]);
    assert_eq!(variables[1].gradient(), [0.0, 1.0]);
    let constant = DualNumber::constant(2.0).unwrap();
    assert!(constant.plus(&constant).unwrap().gradient().is_empty());
    assert_eq!(
        constant.product(&variables[0]).unwrap().gradient(),
        [2.0, 0.0]
    );
    assert_eq!(
        variables[0].product(&constant).unwrap().gradient(),
        [2.0, 0.0]
    );
    let residual = variables[0]
        .product(&variables[0])
        .unwrap()
        .plus(&variables[1].product(&variables[1]).unwrap())
        .unwrap()
        .shifted(-25.0)
        .unwrap();
    assert_eq!(residual.value(), 0.0);
    assert_eq!(residual.gradient(), [6.0, 8.0]);
    assert_eq!(variables[0].gradient(), [1.0, 0.0]);
    assert!(DualNumber::seed(&[]).unwrap().is_empty());
}

#[test]
fn nonsmooth_zero_conventions_and_negative_absolute_match_go() {
    let zero = DualNumber::seed(&[0.0, 0.0]).unwrap();
    for result in [
        zero[0].sqrt(),
        zero[0].abs(),
        zero[0].hypot(&zero[1]),
        zero[0].atan2(&zero[1]),
    ] {
        let result = result.unwrap();
        assert_eq!(result.value(), 0.0);
        assert_eq!(result.gradient(), [0.0, 0.0]);
    }
    let negative = DualNumber::variable(-2.0, 0, 1).unwrap().abs().unwrap();
    assert_eq!(negative.value(), 2.0);
    assert_eq!(negative.gradient(), [-1.0]);
}

#[test]
fn invalid_seeds_dimensions_domains_and_overflow_return_errors() {
    for (index, count) in [(0, 0), (2, 2)] {
        let error = DualNumber::variable(1.0, index, count).unwrap_err();
        assert!(error.to_string().contains(&format!("index {index}")));
    }
    for value in [f64::INFINITY, f64::NEG_INFINITY, f64::NAN] {
        assert!(DualNumber::constant(value).is_err());
        assert!(DualNumber::seed(&[value]).is_err());
    }
    let a = DualNumber::variable(1.0, 0, 1).unwrap();
    let b = DualNumber::variable(1.0, 0, 2).unwrap();
    assert!(a.plus(&b).unwrap_err().to_string().contains("lengths 1, 2"));
    assert!(a.quotient(&DualNumber::constant(0.0).unwrap()).is_err());
    assert!(a.negated().unwrap().sqrt().is_err());
    assert!(a.scaled(f64::INFINITY).is_err());
    assert!(a.shifted(f64::NAN).is_err());
    let large = DualNumber::constant(f64::MAX).unwrap();
    assert!(large.product(&large).is_err());
    let tiny = DualNumber::seed(&[1e-320, 1e-320]).unwrap();
    assert!(tiny[0].quotient(&tiny[1]).is_err());
}

#[test]
fn extreme_scale_angles_and_quotients_keep_representable_derivatives() {
    for scale in [1e-200, 1e200, f64::MAX] {
        let variables = DualNumber::seed(&[scale, scale]).unwrap();
        let angle = variables[0].atan2(&variables[1]).unwrap();
        close(angle.value(), std::f64::consts::FRAC_PI_4, 0.0);
        close(angle.gradient()[0] * scale, 0.5, 3e-16);
        close(angle.gradient()[1] * scale, -0.5, 3e-16);
        let ratio = variables[0].quotient(&variables[1]).unwrap();
        close(ratio.value(), 1.0, 0.0);
        close(ratio.gradient()[0] * scale, 1.0, 3e-16);
        close(ratio.gradient()[1] * scale, -1.0, 3e-16);
    }
}
