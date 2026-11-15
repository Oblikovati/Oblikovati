// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use num_bigint::BigInt;
use num_rational::BigRational;
use oblikovati_model::sketch::conditional_parameter_bound;

fn rational(value: f64) -> BigRational {
    BigRational::from_float(value).unwrap()
}
fn ratio(numerator: i64, denominator: BigInt) -> BigRational {
    BigRational::new(BigInt::from(numerator), denominator)
}
fn bound_for(source: &str) -> ([f64; 2], ParameterCollection<u64>) {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "size", USER_PARAM, source)
        .unwrap();
    (
        conditional_parameter_bound(&parameters, 1)
            .unwrap()
            .bounds(),
        parameters,
    )
}
#[test]
fn authored_decimal_bounds_contain_exact_physical_values() {
    for (source, exact) in [
        ("10 mm", ratio(1, BigInt::from(1))),
        ("1.25 cm", ratio(5, BigInt::from(4))),
        ("0.01 m", ratio(1, BigInt::from(1))),
        ("0.1 mm", ratio(1, BigInt::from(100))),
        ("-0.0 cm", ratio(0, BigInt::from(1))),
        ("1.25e2 mm", ratio(25, BigInt::from(2))),
        ("1e-320 mm", ratio(1, BigInt::from(10).pow(321))),
        ("1e-400 mm", ratio(1, BigInt::from(10).pow(401))),
        ("5e-324 cm", ratio(5, BigInt::from(10).pow(324))),
    ] {
        let (bounds, parameters) = bound_for(source);
        assert!(
            rational(bounds[0]) <= exact && exact <= rational(bounds[1]),
            "{source}: {bounds:?}"
        );
        let rounded = parameters.model_value(1).unwrap().value;
        assert!(bounds[0] <= rounded && rounded <= bounds[1]);
    }
}
#[test]
fn alias_provenance_is_retained_and_unsupported_inputs_decline() {
    let (_, mut parameters) = bound_for("10 mm");
    parameters
        .insert_expression(2, "width", USER_PARAM, "size")
        .unwrap();
    let bound = conditional_parameter_bound(&parameters, 2).unwrap();
    assert_eq!(bound.sources(), &[(2, "size".into()), (1, "10 mm".into())]);
    parameters.set_expression(2, "size / 2").unwrap();
    assert!(conditional_parameter_bound(&parameters, 2).is_err());
    parameters.set_expression(2, "size").unwrap();
    let _ = parameters.set_expression(1, "width");
    assert!(conditional_parameter_bound(&parameters, 1).is_err());
}
#[test]
fn unsupported_units_scales_and_model_tolerances_decline() {
    let (_, mut parameters) = bound_for("10 mm");
    parameters.set_expression(1, "1 in").unwrap();
    assert!(conditional_parameter_bound(&parameters, 1).is_err());
    parameters.set_expression(1, "10 mm").unwrap();
    parameters
        .set_model_value_type(1, oblikovati_api::types::MODEL_VALUE_UPPER)
        .unwrap();
    assert!(conditional_parameter_bound(&parameters, 1).is_err());
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    let mut scaled = ParameterCollection::new(units);
    scaled
        .insert_expression(1, "size", USER_PARAM, "10 mm")
        .unwrap();
    assert!(conditional_parameter_bound(&scaled, 1).is_err());
}

#[test]
fn overflow_literals_are_never_treated_as_certifiable_lengths() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    assert!(
        parameters
            .insert_expression(1, "overflow", USER_PARAM, "1e309 mm")
            .is_err()
    );
    assert!(conditional_parameter_bound(&parameters, 1).is_err());
}
