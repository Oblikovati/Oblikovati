// SPDX-License-Identifier: GPL-2.0-only

use super::*;

/// Read the canonical category spelling; for example `kind_name(USER_PARAM) == Some("user")`.
pub fn kind_name(kind: ParameterKind) -> Option<&'static str> {
    match kind {
        USER_PARAM => Some("user"),
        MODEL_PARAM => Some("model"),
        TABLE_PARAM => Some("table"),
        REFERENCE_PARAM => Some("reference"),
        DERIVED_PARAM => Some("derived"),
        _ => None,
    }
}

impl<Id: Debug> std::fmt::Display for ParameterFailure<Id> {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Unresolved(names) => write!(
                out,
                "parameter names {names:?}: expected resolved parameters"
            ),
            Self::ExternalLink {
                source_document,
                source_parameter,
            } => write!(
                out,
                "external parameter {source_document:?}/{source_parameter:?}: expected an available source"
            ),
            Self::Cycle(error) => std::fmt::Display::fmt(error, out),
            Self::Evaluation(error) => std::fmt::Display::fmt(error, out),
            Self::BlockedBy(ids) => write!(
                out,
                "parameter dependencies {ids:?}: expected healthy values"
            ),
            Self::NonFinite(quantity) => write!(
                out,
                "parameter value {quantity:?}: expected a finite quantity"
            ),
            Self::NonNumeric(ids) => write!(
                out,
                "parameter dependencies {ids:?}: expected numeric values"
            ),
        }
    }
}
