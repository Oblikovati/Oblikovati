// SPDX-License-Identifier: GPL-2.0-only
//! Observe each completed feature boundary during ordered replay.
use super::evaluate::require_replay_current;
use super::{
    PartFeatureDefinition, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    PartFeatureProgramError,
};
use crate::feature::{FeatureHealthRecord, FeatureReplayController, FeatureReplayStep};
use crate::history::DocumentSnapshot;
use std::borrow::Cow;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn replay_observed<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &mut Cow<'_, DocumentSnapshot>,
        backend: &mut B,
        controller: &mut FeatureReplayController<
            '_,
            PartFeatureDefinition,
            PartFeatureFailure<Failure>,
        >,
        options: Option<oblikovati_solve::nonlinear::SolveOptions>,
        cancelled: &mut impl FnMut() -> bool,
        observe: &mut impl FnMut(&Self, &[FeatureHealthRecord<PartFeatureFailure<Failure>>]),
    ) -> Result<(), PartFeatureProgramError> {
        loop {
            require_replay_current(cancelled(), controller.records().len())?;
            let Some(step) = controller
                .advance()
                .map_err(PartFeatureProgramError::Reads)?
            else {
                return Ok(());
            };
            if let FeatureReplayStep::Evaluate(id) = step {
                self.refresh_sketch_references(snapshot, backend);
                if let Some(options) = options {
                    self.solve_feature_sketch(
                        snapshot,
                        id,
                        options,
                        controller.records().len(),
                        cancelled,
                    )?;
                }
                let outcome = self.evaluate_entry(snapshot, backend, id);
                controller
                    .record_result(id, outcome)
                    .map_err(PartFeatureProgramError::Reads)?;
            }
            observe(self, controller.records());
        }
    }
}
