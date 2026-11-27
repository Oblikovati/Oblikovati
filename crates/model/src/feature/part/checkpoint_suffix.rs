// SPDX-License-Identifier: GPL-2.0-only
//! Build a new checkpoint set from an unchanged prefix and a replayed tail.
use super::{PartFeatureEvaluator, PartReplayCheckpoints, checkpoints::copy_state};
use crate::{
    feature::{FeatureReplayController, FeatureReplayRange},
    history::DocumentSnapshot,
};
use std::{borrow::Cow, sync::Arc};

impl<Body: Clone, Failure: Clone> PartReplayCheckpoints<Body, Failure> {
    /// Replay a changed tail and retain its new feature boundaries.
    /// Example: `old.replay_with_checkpoints(&edited, &mut backend, || false)?`.
    /// The caller must validate the source generation, policy, and sketch inputs.
    pub fn replay_with_checkpoints<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &self,
        candidate: &DocumentSnapshot,
        backend: &mut B,
        mut cancelled: impl FnMut() -> bool,
    ) -> Result<Self, String> {
        let change = self
            .result()
            .replay_change(candidate)
            .ok_or("part checkpoints: expected changed evaluated inputs")?;
        let prefix = self.states.get(change.start).ok_or_else(|| {
            format!(
                "part checkpoint {}: expected a retained prefix boundary",
                change.start
            )
        })?;
        let range = candidate
            .part_replay_range()
            .map_err(|error| error.to_string())?;
        if change.start > range.end {
            return Err(format!(
                "part checkpoint {}, end {}: expected a valid replay range",
                change.start, range.end
            ));
        }
        let sequence =
            super::body_dependencies::with_replacement_reads(candidate.effective_work_geometry())
                .map_err(|error| error.to_string())?;
        let mut controller = FeatureReplayController::new_with_parameters(
            &sequence,
            FeatureReplayRange {
                start: change.start,
                end: range.end,
            },
            prefix.health.clone(),
            &candidate.parameters,
        )
        .map_err(|error| error.to_string())?;
        let source = Arc::new(candidate.clone());
        let mut states = self.states[..=change.start].to_vec();
        let mut result = prefix.clone();
        let mut candidate = Cow::Borrowed(candidate);
        result
            .replay_observed(
                &mut candidate,
                backend,
                &mut controller,
                None,
                &mut cancelled,
                &mut |state, health| states.push(copy_state(state, health, &source)),
            )
            .map_err(|error| error.to_string())?;
        result.refresh_sketch_references(&candidate, backend);
        result.health = controller.finish().map_err(|error| error.to_string())?;
        if cancelled() {
            return Err(format!(
                "part checkpoint replay after {} features: expected a current generation",
                result.health.len()
            ));
        }
        for state in &mut states {
            state.source = Some(Arc::clone(&source));
        }
        result.source = Some(source);
        *states.last_mut().expect("prefix boundary exists") = result;
        Ok(Self { states })
    }
}
