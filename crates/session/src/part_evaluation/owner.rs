// SPDX-License-Identifier: GPL-2.0-only
//! Shared document, history, and generation checks for derived part results.
use super::{DocumentSession, PartEvaluationError, TransactionObserver};
use oblikovati_model::history::EvaluationScope;

pub(super) fn validate_owner<O: TransactionObserver>(
    document: u64,
    scope: &EvaluationScope,
    session: &DocumentSession<O>,
) -> Result<(), PartEvaluationError> {
    if document != session.document {
        return Err(PartEvaluationError::ForeignDocument {
            supplied: document,
            expected: session.document,
        });
    }
    if !scope.belongs_to(&session.history) {
        return Err(PartEvaluationError::ForeignHistory {
            document,
            generation: scope.generation(),
        });
    }
    Ok(())
}
pub(super) fn validate_current<O: TransactionObserver>(
    document: u64,
    scope: &EvaluationScope,
    session: &DocumentSession<O>,
) -> Result<(), PartEvaluationError> {
    validate_owner(document, scope, session)?;
    if !scope.is_current(&session.history) {
        return Err(PartEvaluationError::Superseded {
            document,
            supplied: scope.generation(),
            expected: session.generation(),
        });
    }
    Ok(())
}
