// SPDX-License-Identifier: GPL-2.0-only

use crate::{dependency::DependencyError, expression::ExpressionError, quantity::Quantity};
use oblikovati_api::types::{ModelValueType, ParameterKind, Tolerance};

/// Application-owned values; this is not a wire DTO or persisted encoding.
#[derive(Clone, Debug, PartialEq)]
pub enum ParameterValue {
    Numeric(Quantity),
    Text(String),
    Boolean(bool),
}

/// A numeric parameter's evaluation failure, separate from entity health in the API.
#[derive(Clone, Debug, PartialEq)]
pub enum ParameterFailure<Id> {
    Unresolved(Vec<String>),
    ExternalLink {
        source_document: String,
        source_parameter: String,
    },
    Cycle(DependencyError<Id>),
    Evaluation(ExpressionError),
    BlockedBy(Vec<Id>),
    NonFinite(Quantity),
    NonNumeric(Vec<Id>),
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
    pub annotations: super::ParameterAnnotations,
    pub kind: ParameterKind,
    pub renamed: bool,
    pub built_in: bool,
    pub disabled_actions: oblikovati_api::types::ActionType,
    pub custom_property: super::CustomPropertyFormat,
    pub tolerance: Tolerance,
    pub choices: oblikovati_api::wire::ExpressionListInfo,
    pub model_value_type: ModelValueType,
    pub source: String,
    pub expression_context: Option<super::DimensionExpressionContext>,
    pub last_good_value: Option<ParameterValue>,
    pub health: ParameterHealth<Id>,
}

impl<Id> ParameterState<Id> {
    /// Read a usable value only when evaluation succeeded; for example `parameter.current_value()`.
    pub fn current_value(&self) -> Option<Quantity> {
        match self.current() {
            Some(ParameterValue::Numeric(quantity)) => Some(*quantity),
            _ => None,
        }
    }

    /// Read a healthy typed value; for example `parameter.current()`.
    pub fn current(&self) -> Option<&ParameterValue> {
        match self.health {
            ParameterHealth::Healthy => self.last_good_value.as_ref(),
            ParameterHealth::Failed(_) => None,
        }
    }
}
