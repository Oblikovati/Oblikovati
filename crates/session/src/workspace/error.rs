// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::IdentityRegistrationError;
use std::{error::Error, fmt};

/// A refused workspace operation, leaving existing ownership and focus intact.
#[derive(Debug, PartialEq, Eq)]
pub enum WorkspaceError {
    Missing(u64),
    NoActive,
    Exhausted,
    Identity(IdentityRegistrationError),
    InitialVerification(oblikovati_model::history::HistoryError),
    CloseVeto { document: u64, reason: String },
}
impl From<IdentityRegistrationError> for WorkspaceError {
    fn from(error: IdentityRegistrationError) -> Self {
        Self::Identity(error)
    }
}
impl fmt::Display for WorkspaceError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Missing(id) => write!(output, "document {id}: expected an open document"),
            Self::NoActive => write!(
                output,
                "active document is absent: expected an activated open document"
            ),
            Self::Exhausted => write!(output, "session ID u64::MAX: expected an unused ID"),
            Self::Identity(error) => error.fmt(output),
            Self::InitialVerification(error) => {
                write!(output, "initial sketch verification: {error}")
            }
            Self::CloseVeto { document, reason } => write!(
                output,
                "close document {document} veto {reason:?}: expected policy approval"
            ),
        }
    }
}
impl Error for WorkspaceError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Identity(error) => Some(error),
            Self::InitialVerification(error) => Some(error),
            _ => None,
        }
    }
}
