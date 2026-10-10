// SPDX-License-Identifier: GPL-2.0-only

use crate::{dependency::DependencyError, expression::ExpressionError};
use oblikovati_api::types::ParameterKind;
use std::fmt;

/// Rejected collection edit. Evaluation failures instead live on parameter health.
#[derive(Clone, Debug, PartialEq)]
pub enum ParameterError<Id> {
    Dependency(DependencyError<Id>),
    Syntax(ExpressionError),
    Value {
        id: Id,
        found: String,
        expected: &'static str,
    },
    Name {
        name: String,
        expected: &'static str,
    },
    Kind {
        kind: ParameterKind,
        expected: &'static str,
    },
    ReadOnly {
        id: Id,
        name: String,
    },
    Group {
        key: String,
        found: String,
        expected: &'static str,
    },
    InUse {
        id: Id,
        readers: Vec<String>,
    },
}

impl<Id: fmt::Debug> fmt::Display for ParameterError<Id> {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Dependency(error) => error.fmt(formatter),
            Self::Syntax(error) => error.fmt(formatter),
            Self::Value {
                id,
                found,
                expected,
            } => write!(
                formatter,
                "parameter {id:?}: found {found}; expected {expected}"
            ),
            Self::Name { name, expected } => {
                write!(formatter, "parameter name {name:?}: expected {expected}")
            }
            Self::Kind { kind, expected } => {
                write!(formatter, "parameter kind {kind:?}: expected {expected}")
            }
            Self::ReadOnly { id, name } => write!(
                formatter,
                "parameter {name:?} ({id:?}): expected an editable parameter"
            ),
            Self::Group {
                key,
                found,
                expected,
            } => write!(
                formatter,
                "group {key:?}: found {found:?}; expected {expected}"
            ),
            Self::InUse { id, readers } => write!(
                formatter,
                "parameter {id:?} is used by {readers:?}: expected no readers or owning feature"
            ),
        }
    }
}
impl<Id: fmt::Debug> std::error::Error for ParameterError<Id> {}

impl<Id> From<DependencyError<Id>> for ParameterError<Id> {
    fn from(error: DependencyError<Id>) -> Self {
        Self::Dependency(error)
    }
}
impl<Id> From<ExpressionError> for ParameterError<Id> {
    fn from(error: ExpressionError) -> Self {
        Self::Syntax(error)
    }
}
