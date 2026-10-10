// SPDX-License-Identifier: GPL-2.0-only

use super::{Expression, ExpressionError};
use crate::quantity::Quantity;
use std::collections::BTreeSet;

/// An expression bound to application IDs; unresolved names remain visible to the owner.
#[derive(Clone, Debug)]
pub struct BoundExpression<Id> {
    pub(super) expression: Expression,
    pub(super) bindings: Vec<Option<Id>>,
}

impl<Id: Copy + Ord> BoundExpression<Id> {
    /// Read unresolved names in source order; for example `bound.unresolved()`.
    pub fn unresolved(&self) -> Vec<&str> {
        self.expression
            .references
            .iter()
            .zip(&self.bindings)
            .filter(|(_, id)| id.is_none())
            .map(|(name, _)| name.as_str())
            .collect()
    }

    /// Read distinct bound IDs in source order; for example `graph.replace_drivers(id, bound.bound_ids())`.
    pub fn bound_ids(&self) -> Vec<Id> {
        let mut seen = BTreeSet::new();
        self.bindings
            .iter()
            .flatten()
            .copied()
            .filter(|id| seen.insert(*id))
            .collect()
    }

    /// Evaluate using current values without rebinding names; for example `bound.evaluate(|id| scope.value_of(id))`.
    pub fn evaluate(
        &self,
        mut value_of: impl FnMut(Id) -> Option<Quantity>,
    ) -> Result<Quantity, ExpressionError> {
        self.expression.evaluate_with(|index| {
            let id = self.bindings[index]
                .ok_or_else(|| self.reference_error(index, "a binding to a stable parameter ID"))?;
            value_of(id).ok_or_else(|| {
                self.reference_error(index, "a current value for the bound parameter ID")
            })
        })
    }

    fn reference_error(&self, index: usize, expected: &'static str) -> ExpressionError {
        ExpressionError::Reference {
            name: self.expression.references[index].clone(),
            expected,
        }
    }
}
