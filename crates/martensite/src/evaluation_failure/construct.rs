// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl EvaluationPhase {
    pub(crate) fn refused(
        self,
        kind: EvaluationFailureKind,
        entity: EvaluationEntity,
        message: String,
    ) -> EvaluationFailure {
        EvaluationFailure(Box::new(FailureDetails {
            phase: self,
            kind,
            entity,
            feature: None,
            sketch: None,
            prefix: None,
            message,
            cause: None,
        }))
    }
    pub(crate) fn native(
        self,
        entity: EvaluationEntity,
        message: String,
        cause: Failure,
    ) -> EvaluationFailure {
        let mut failure = self.refused(EvaluationFailureKind::Kernel, entity, message);
        failure.0.cause = Some(EvaluationCause::Kernel(cause));
        failure
    }
    pub(crate) fn sketch(self, entity: EvaluationEntity, cause: SketchError) -> EvaluationFailure {
        let mut failure = self.refused(EvaluationFailureKind::InvalidInput, entity, String::new());
        failure.0.cause = Some(EvaluationCause::Sketch(cause));
        failure
    }
}
