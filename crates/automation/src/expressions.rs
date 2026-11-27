// SPDX-License-Identifier: GPL-2.0-only

use crate::units::require_category;
use oblikovati_api::{Error, wire::*};
use oblikovati_model::{
    expression::Expression,
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::{UnitsOfMeasure, category_name},
};

pub(super) fn value(
    parameters: &ParameterCollection<u64>,
    request: ExpressionWithTypeArgs,
) -> Result<ValueResult, Error> {
    let category = require_category(&request.units_type)?;
    let mut quantity = evaluate(parameters, &request.expression)?;
    // Preserve the Go wire convention: only bare values use the requested display unit.
    if quantity.dimension == Dimension::Unitless && category != Dimension::Unitless {
        quantity = parameters.units().from_preferred(quantity.value, category);
    }
    Ok(ValueResult {
        value: quantity.value,
    })
}

pub(super) fn database(
    parameters: &ParameterCollection<u64>,
    request: ExpressionArgs,
) -> Result<DatabaseUnitsResult, Error> {
    let quantity = evaluate(parameters, &request.expression)?;
    Ok(DatabaseUnitsResult {
        value: quantity.value,
        units_type: category_name(quantity.dimension).into(),
    })
}

pub(super) fn valid(
    parameters: &ParameterCollection<u64>,
    request: ExpressionWithTypeArgs,
) -> Result<ExpressionValidResult, Error> {
    let category = require_category(&request.units_type)?;
    let failure = match evaluate(parameters, &request.expression) {
        Ok(quantity) if compatible_category(quantity.dimension, category) => String::new(),
        Ok(quantity) => format!(
            "expression unit {} is not compatible with {}",
            category_name(quantity.dimension),
            category_name(category)
        ),
        Err(error) => error.to_string(),
    };
    Ok(ExpressionValidResult {
        valid: failure.is_empty(),
        error: failure,
    })
}

pub(super) fn compatible(
    parameters: &ParameterCollection<u64>,
    request: ExpressionWithTypeArgs,
) -> Result<CompatibleUnitsResult, Error> {
    let category = require_category(&request.units_type)?;
    Ok(CompatibleUnitsResult {
        compatible: evaluate(parameters, &request.expression)
            .is_ok_and(|quantity| compatible_category(quantity.dimension, category)),
    })
}

fn compatible_category(actual: Dimension, expected: Dimension) -> bool {
    actual == expected || actual == Dimension::Unitless
}

fn evaluate(parameters: &ParameterCollection<u64>, source: &str) -> Result<Quantity, Error> {
    parameters
        .evaluate_expression(source)
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn references(request: ExpressionArgs) -> Result<DrivingParametersResult, Error> {
    let expression = parse(&request.expression)?;
    Ok(DrivingParametersResult {
        names: Some(expression.references().to_vec()),
    })
}

pub(super) fn canonical(request: ExpressionArgs) -> Result<StringResult, Error> {
    parse(&request.expression)?;
    Ok(StringResult {
        value: request.expression,
    })
}

fn parse(source: &str) -> Result<Expression, Error> {
    // These syntax-only queries do not require an open document or resolve names.
    Expression::parse(source, &UnitsOfMeasure::default()).map_err(|error| Error(error.to_string()))
}
