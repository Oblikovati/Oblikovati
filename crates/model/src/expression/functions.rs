// SPDX-License-Identifier: GPL-2.0-only

use super::ExpressionError;
use crate::quantity::{Dimension, Quantity};
use Dimension::{Angle, Area, Length, Unitless};

#[derive(Clone, Copy)]
enum UnaryRule {
    Trig,
    InverseTrig,
    Preserve,
    Unitless,
    SquareRoot,
}

enum Builtin {
    Unary(fn(f64) -> f64, UnaryRule),
    Minimum,
    Maximum,
    Atan2,
}

type UnaryEntry = (&'static str, fn(f64) -> f64, UnaryRule);

const UNARY_FUNCTIONS: [UnaryEntry; 14] = [
    ("sin", f64::sin, UnaryRule::Trig),
    ("cos", f64::cos, UnaryRule::Trig),
    ("tan", f64::tan, UnaryRule::Trig),
    ("asin", f64::asin, UnaryRule::InverseTrig),
    ("acos", f64::acos, UnaryRule::InverseTrig),
    ("atan", f64::atan, UnaryRule::InverseTrig),
    ("abs", f64::abs, UnaryRule::Preserve),
    ("floor", f64::floor, UnaryRule::Preserve),
    ("ceil", f64::ceil, UnaryRule::Preserve),
    ("round", f64::round, UnaryRule::Preserve),
    ("ln", f64::ln, UnaryRule::Unitless),
    ("log", f64::log10, UnaryRule::Unitless),
    ("exp", f64::exp, UnaryRule::Unitless),
    ("sqrt", f64::sqrt, UnaryRule::SquareRoot),
];

fn lookup(name: &str) -> Option<Builtin> {
    match name {
        "min" => Some(Builtin::Minimum),
        "max" => Some(Builtin::Maximum),
        "atan2" => Some(Builtin::Atan2),
        _ => UNARY_FUNCTIONS
            .iter()
            .find(|(spelling, _, _)| *spelling == name)
            .map(|(_, function, rule)| Builtin::Unary(*function, *rule)),
    }
}

pub(super) fn require_known(name: &str) -> Result<(), ExpressionError> {
    lookup(name)
        .map(|_| ())
        .ok_or_else(|| failure(name, name, "a registered built-in function"))
}

pub(super) fn evaluate(name: &str, arguments: &[Quantity]) -> Result<Quantity, ExpressionError> {
    let builtin =
        lookup(name).ok_or_else(|| failure(name, name, "a registered built-in function"))?;
    match builtin {
        Builtin::Unary(function, rule) => unary(name, arguments, function, rule),
        Builtin::Minimum => extreme(name, arguments, true),
        Builtin::Maximum => extreme(name, arguments, false),
        Builtin::Atan2 => atan2(name, arguments),
    }
}

fn unary(
    name: &str,
    arguments: &[Quantity],
    function: fn(f64) -> f64,
    rule: UnaryRule,
) -> Result<Quantity, ExpressionError> {
    if arguments.len() != 1 {
        return Err(failure(name, arguments.len(), "one argument"));
    }
    let argument = arguments[0];
    let dimension = unary_dimension(rule, argument.dimension).ok_or_else(|| {
        failure(
            name,
            format!("{:?}", argument.dimension),
            dimension_requirement(rule),
        )
    })?;
    Ok(Quantity::new(function(argument.value), dimension))
}

fn unary_dimension(rule: UnaryRule, input: Dimension) -> Option<Dimension> {
    match rule {
        UnaryRule::Trig if matches!(input, Angle | Unitless) => Some(Unitless),
        UnaryRule::InverseTrig if input == Unitless => Some(Angle),
        UnaryRule::Unitless if input == Unitless => Some(Unitless),
        UnaryRule::Preserve => Some(input),
        UnaryRule::SquareRoot if input == Area => Some(Length),
        UnaryRule::SquareRoot if input == Unitless => Some(Unitless),
        _ => None,
    }
}

fn dimension_requirement(rule: UnaryRule) -> &'static str {
    match rule {
        UnaryRule::Trig => "an Angle or Unitless argument",
        UnaryRule::InverseTrig | UnaryRule::Unitless => "a Unitless argument",
        UnaryRule::SquareRoot => "an Area or Unitless argument with a named square-root dimension",
        UnaryRule::Preserve => "a quantity",
    }
}

fn extreme(name: &str, arguments: &[Quantity], minimum: bool) -> Result<Quantity, ExpressionError> {
    if arguments.len() < 2 {
        return Err(failure(name, arguments.len(), "two or more arguments"));
    }
    require_same_dimensions(name, arguments)?;
    let value = arguments[1..]
        .iter()
        .fold(arguments[0].value, |left, right| {
            ieee_extreme(left, right.value, minimum)
        });
    Ok(Quantity::new(value, arguments[0].dimension))
}

fn ieee_extreme(left: f64, right: f64, minimum: bool) -> f64 {
    // Go math/dim.go gives the selected infinity precedence over NaN. Rust
    // f64::min/max ignore NaN; neither Rust primitive implements this contract.
    let extreme = if minimum {
        f64::NEG_INFINITY
    } else {
        f64::INFINITY
    };
    if left == extreme || right == extreme {
        return extreme;
    }
    if left.is_nan() || right.is_nan() {
        return f64::NAN;
    }
    if left == 0.0 && right == 0.0 {
        return signed_zero(left, right, minimum);
    }
    if (left < right) == minimum {
        left
    } else {
        right
    }
}

fn signed_zero(left: f64, right: f64, minimum: bool) -> f64 {
    let negative = if minimum {
        left.is_sign_negative() || right.is_sign_negative()
    } else {
        left.is_sign_negative() && right.is_sign_negative()
    };
    if negative { -0.0 } else { 0.0 }
}

fn atan2(name: &str, arguments: &[Quantity]) -> Result<Quantity, ExpressionError> {
    if arguments.len() != 2 {
        return Err(failure(name, arguments.len(), "two arguments"));
    }
    require_same_dimensions(name, arguments)?;
    Ok(Quantity::new(
        arguments[0].value.atan2(arguments[1].value),
        Angle,
    ))
}

fn require_same_dimensions(name: &str, arguments: &[Quantity]) -> Result<(), ExpressionError> {
    let dimension = arguments[0].dimension;
    if arguments
        .iter()
        .any(|argument| argument.dimension != dimension)
    {
        return Err(failure(
            name,
            format!("{arguments:?}"),
            "arguments with the same dimension",
        ));
    }
    Ok(())
}

fn failure(name: &str, found: impl ToString, expected: &'static str) -> ExpressionError {
    ExpressionError::Function {
        name: name.into(),
        found: found.to_string(),
        expected,
    }
}
