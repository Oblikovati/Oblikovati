// SPDX-License-Identifier: GPL-2.0-only

use crate::{dependency::DependencyError, expression::ExpressionError, quantity::Quantity};
use oblikovati_api::types::ParameterKind;

/// A numeric parameter's evaluation failure, separate from entity health in the API.
#[derive(Clone, Debug, PartialEq)]
pub enum ParameterFailure<Id> {
    Unresolved(Vec<String>),
    Cycle(DependencyError<Id>),
    Evaluation(ExpressionError),
    BlockedBy(Vec<Id>),
    NonFinite(Quantity),
}

/// Evaluation status. A failed parameter may retain a last-good value for inspection.
#[derive(Clone, Debug, PartialEq)]
pub enum ParameterHealth<Id> {
    Healthy,
    Failed(ParameterFailure<Id>),
}

/// Read-only projection of a stored parameter; edits go through its collection.
#[derive(Clone, Debug, PartialEq)]
pub struct ParameterState<Id> {
    pub name: String,
    pub kind: ParameterKind,
    pub renamed: bool,
    pub source: String,
    pub last_good_value: Option<Quantity>,
    pub health: ParameterHealth<Id>,
}

impl<Id> ParameterState<Id> {
    /// Read a usable value only when evaluation succeeded; for example `parameter.current_value()`.
    pub fn current_value(&self) -> Option<Quantity> {
        match self.health {
            ParameterHealth::Healthy => self.last_good_value,
            ParameterHealth::Failed(_) => None,
        }
    }
}
