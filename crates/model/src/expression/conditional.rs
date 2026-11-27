// SPDX-License-Identifier: GPL-2.0-only
//! Conditional interval evaluation over the same syntax used by model formulas.
use super::{
    node::{BinaryOperator, Node},
    parser,
};
use crate::{
    quantity::{Dimension, Quantity},
    sketch::interval_equations::enclosure::Enclosure,
    units::UnitsOfMeasure,
};

#[derive(Clone, Copy)]
pub(crate) struct ConditionalValue {
    pub bounds: Enclosure,
    pub dimension: Dimension,
}

pub(crate) fn conditional_expression(
    source: &str,
    units: &UnitsOfMeasure,
    mut reference: impl FnMut(&str) -> Result<ConditionalValue, String>,
) -> Result<ConditionalValue, String> {
    let (expression, spans) =
        parser::compile_unfolded(source, units).map_err(|error| error.to_string())?;
    let mut values: Vec<ConditionalValue> = Vec::with_capacity(expression.nodes.len());
    for (index, node) in expression.nodes.iter().enumerate() {
        let value = match node {
            Node::Literal(quantity) => literal(source, spans[index], quantity.dimension)?,
            Node::Reference(index) => reference(&expression.references[*index])?,
            Node::Negate(child) => ConditionalValue {
                bounds: values[*child].bounds.negate(),
                dimension: values[*child].dimension,
            },
            Node::Binary(operator, left, right) => {
                binary(*operator, values[*left], values[*right])?
            }
            Node::Call(name, _) => {
                return Err(format!(
                    "formula function {name:?}: expected an interval-supported operator"
                ));
            }
        };
        values.push(value);
    }
    Ok(values[expression.root])
}

fn literal(
    source: &str,
    span: Option<(usize, usize)>,
    dimension: Dimension,
) -> Result<ConditionalValue, String> {
    let (start, end) =
        span.ok_or_else(|| format!("literal in {source:?}: expected an authored source span"))?;
    let text = &source[start..end];
    let end = crate::units::numeric_prefix_len(text);
    let (number, suffix) = text.split_at(end);
    if number.len() > 128 {
        return Err(format!(
            "literal {text:?}: expected at most 128 numeric characters"
        ));
    }
    let factor = match suffix.trim() {
        "" => "1",
        "mm" => "0.1",
        "cm" => "1",
        "m" => "100",
        _ => return Err(format!("literal {text:?}: expected unitless, mm, cm, or m")),
    };
    Ok(ConditionalValue {
        bounds: Enclosure::decimal(number)?.product(Enclosure::decimal(factor)?)?,
        dimension,
    })
}

fn binary(
    operator: BinaryOperator,
    left: ConditionalValue,
    right: ConditionalValue,
) -> Result<ConditionalValue, String> {
    let dimensions = operator
        .apply(
            Quantity::new(1., left.dimension),
            Quantity::new(1., right.dimension),
        )
        .map_err(|error| error.to_string())?;
    let bounds = match operator {
        BinaryOperator::Add => left.bounds.sum(right.bounds),
        BinaryOperator::Subtract => left.bounds.difference(right.bounds),
        BinaryOperator::Multiply => left.bounds.product(right.bounds),
        BinaryOperator::Divide => left.bounds.quotient(right.bounds),
    }?;
    Ok(ConditionalValue {
        bounds,
        dimension: dimensions.dimension,
    })
}
