// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{expression::ExpressionError, quantity::Quantity};

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Evaluate against healthy nominal values without an edit; for example `parameters.evaluate_expression("width * 2")`.
    pub fn evaluate_expression(&self, source: &str) -> Result<Quantity, ExpressionError> {
        let expression = Expression::parse(source, &self.units)?;
        self.evaluate_compiled_expression(expression)
    }

    pub(super) fn evaluate_compiled_expression(
        &self,
        expression: Expression,
    ) -> Result<Quantity, ExpressionError> {
        let source = expression.source();
        let value = expression
            .bind(|name| self.find(name))
            .evaluate(|id| self.records[&id].state.current_value())?;
        if !value.value.is_finite() {
            return Err(ExpressionError::NonFinite {
                source: source.into(),
                value,
            });
        }
        Ok(value)
    }
}
