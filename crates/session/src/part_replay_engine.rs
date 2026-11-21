// SPDX-License-Identifier: GPL-2.0-only
//! Run and adopt named-part recomputes on the document owner.
use crate::{DocumentSession, TransactionObserver};
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid, PrismResolutionPolicy,
};
use oblikovati_model::identity::DocumentGuid;

type SketchRecompute =
    crate::part_evaluation::CompletedPartSketchRecompute<NamedSolid, EvaluationFailure>;

impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn recompute_owned_part(
        &mut self,
        document: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        if self.snapshot().requires_named_part_replay() {
            let completed = self.recompute_owned_sketches(document, policy, attempts)?;
            if self.adopt_owned_sketches(policy, completed)? {
                return Ok(());
            }
        }
        self.recompute_owned_full_part(document, policy, attempts)
    }

    fn recompute_owned_sketches(
        &self,
        document: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<SketchRecompute, String> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(self.history(), document, policy);
        *attempts = attempts.saturating_add(1);
        self.prepare_part_evaluation()
            .recompute_sketches(&mut evaluator, self.sketch_solve_options())
            .map_err(|error| error.to_string())
    }

    fn recompute_owned_full_part(
        &mut self,
        document: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(self.history(), document, policy);
        *attempts = attempts.saturating_add(1);
        let completed = self
            .prepare_part_evaluation()
            .evaluate(&mut evaluator)
            .map_err(|error| error.to_string())?;
        self.adopt_owned_part(policy, completed)
    }
}
