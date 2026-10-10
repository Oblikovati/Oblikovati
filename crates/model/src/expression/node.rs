// SPDX-License-Identifier: GPL-2.0-only

use super::{ExpressionError, functions};
use crate::quantity::Quantity;

#[derive(Clone, Copy, Debug)]
pub(super) enum BinaryOperator {
    Add,
    Subtract,
    Multiply,
    Divide,
}

impl BinaryOperator {
    pub fn apply(self, left: Quantity, right: Quantity) -> Result<Quantity, ExpressionError> {
        let result = match self {
            Self::Add => left.checked_add(right),
            Self::Subtract => left.checked_sub(right),
            Self::Multiply => left.checked_mul(right),
            Self::Divide => left.checked_div(right),
        };
        result.map_err(Into::into)
    }
}

#[derive(Clone, Debug)]
pub(super) enum Node {
    Literal(Quantity),
    Reference(usize),
    Negate(usize),
    Binary(BinaryOperator, usize, usize),
    Call(String, Vec<usize>),
}

impl Node {
    pub fn children(&self) -> Vec<usize> {
        match self {
            Self::Literal(_) | Self::Reference(_) => Vec::new(),
            Self::Negate(child) => vec![*child],
            Self::Binary(_, left, right) => vec![*left, *right],
            Self::Call(_, arguments) => arguments.clone(),
        }
    }

    pub fn compute(&self, values: &[Quantity]) -> Result<Quantity, ExpressionError> {
        match self {
            Self::Literal(quantity) => Ok(*quantity),
            Self::Negate(_) => Ok(values[0].negated()),
            Self::Binary(operator, _, _) => operator.apply(values[0], values[1]),
            Self::Call(name, _) => functions::evaluate(name, values),
            Self::Reference(_) => {
                unreachable!("references are resolved through the caller's scope")
            }
        }
    }

    pub fn fold(self, arena: &[Node]) -> Self {
        if matches!(self, Self::Reference(_)) {
            return self;
        }
        let constants: Option<Vec<Quantity>> = self
            .children()
            .iter()
            .map(|index| match arena[*index] {
                Self::Literal(quantity) => Some(quantity),
                _ => None,
            })
            .collect();
        constants
            .and_then(|values| self.compute(&values).ok())
            .map_or(self, Self::Literal)
    }
}
