// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchConstraintId, SketchEntityId, SketchError};
use oblikovati_solve::{nonlinear::SolveError, residual::EvaluationError};

/// Complete source identity; an arc invariant belongs to its entity, not an authored constraint.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchRelationId {
    Authored(SketchConstraintId),
    ArcCircularity(SketchEntityId),
}
impl SketchRelationId {
    pub(super) fn authored(self) -> Option<SketchConstraintId> {
        match self {
            Self::Authored(id) => Some(id),
            Self::ArcCircularity(_) => None,
        }
    }
}
pub(super) fn source_failure(origins: &[SketchRelationId], cause: SolveError) -> SketchError {
    if let SolveError::Evaluation(EvaluationError::Source { block, .. }) = &cause
        && let Some(origin) = origins.get(*block)
        && let SolveError::Evaluation(cause) = cause
    {
        return match *origin {
            SketchRelationId::Authored(constraint) => SketchError::Residual { constraint, cause },
            SketchRelationId::ArcCircularity(arc) => SketchError::ArcCircularity { arc, cause },
        };
    }
    cause.into()
}
