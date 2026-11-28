// SPDX-License-Identifier: GPL-2.0-only
//! Retain feature boundaries through the owning document session.
use crate::{DocumentSession, TransactionObserver};
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::identity::DocumentGuid;

impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn ensure_owned_checkpoints(
        &mut self,
        document: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        if self
            .owned_checkpoint_slot(policy)
            .is_some_and(|slot| slot.candidate_prefix(self).is_ok())
        {
            return Ok(());
        }
        let current = self
            .owned_part_slot(policy)
            .is_some_and(|slot| slot.current(self).is_ok());
        let certified = policy.ok_or("part checkpoints: expected a certified resolution policy")?;
        if self.snapshot().requires_named_part_replay()
            && !current
            && self.recompute_sketch_checkpoints(document, certified, attempts)?
        {
            return Ok(());
        }
        self.recompute_part_checkpoints(document, certified, current, attempts)
    }

    fn recompute_sketch_checkpoints(
        &mut self,
        document: DocumentGuid,
        policy: PrismResolutionPolicy,
        attempts: &mut u64,
    ) -> Result<bool, String> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(self.history(), document, policy)
                .with_derived_sources(&self.derived_sources);
        *attempts = attempts.saturating_add(1);
        let completed = self
            .prepare_part_evaluation()
            .recompute_sketches_with_checkpoints(&mut evaluator, self.sketch_solve_options())
            .map_err(|error| error.to_string())?;
        self.adopt_owned_sketch_checkpoints(Some(policy), completed)
    }

    fn recompute_part_checkpoints(
        &mut self,
        document: DocumentGuid,
        policy: PrismResolutionPolicy,
        current: bool,
        attempts: &mut u64,
    ) -> Result<(), String> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(self.history(), document, policy)
                .with_derived_sources(&self.derived_sources);
        *attempts = attempts.saturating_add(1);
        let completed = self
            .prepare_part_evaluation()
            .evaluate_checkpoints(&mut evaluator)
            .map_err(|error| error.to_string())?;
        if current {
            return self.adopt_owned_checkpoints(Some(policy), completed);
        }
        self.adopt_owned_full_checkpoints(Some(policy), completed)
    }
}
