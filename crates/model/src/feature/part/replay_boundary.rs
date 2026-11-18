// SPDX-License-Identifier: GPL-2.0-only
//! Conservative recipe boundary; callers must separately certify retained outputs.
use crate::{feature::FeatureReplayRange, history::DocumentSnapshot};

impl DocumentSnapshot {
    /// Locate the earliest potentially changed part input, without certifying cached results.
    /// Example: `old.part_replay_change(&edited)` returns the dirty tail, if any.
    pub fn part_replay_change(&self, candidate: &Self) -> Option<FeatureReplayRange> {
        let previous = self.effective_work_geometry();
        let current = candidate.effective_work_geometry();
        let end = current.effective_end();
        if !self.parameters.same_evaluated_state(&candidate.parameters)
            || !self.sketches.same_evaluated_state(&candidate.sketches)
        {
            return Some(FeatureReplayRange { start: 0, end });
        }
        let old_end = previous.effective_end();
        let shared = old_end.min(end);
        let mismatch = previous.features()[..shared]
            .iter()
            .zip(&current.features()[..shared])
            .position(|(old, new)| !old.same_replay_recipe(new));
        let start = mismatch.or_else(|| (old_end != end).then_some(shared))?;
        let unknown = previous.features()[..start]
            .iter()
            .chain(&current.features()[..start])
            .any(|feature| feature.reads().has_unattributed_reads());
        Some(FeatureReplayRange {
            start: if unknown { 0 } else { start },
            end,
        })
    }
}
