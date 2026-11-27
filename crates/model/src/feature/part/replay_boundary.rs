// SPDX-License-Identifier: GPL-2.0-only
//! Conservative recipe boundary; callers must separately certify retained outputs.
use crate::{
    feature::{FeatureReadKey, FeatureReplayRange},
    history::DocumentSnapshot,
};

impl DocumentSnapshot {
    /// Locate the earliest potentially changed part input, without certifying cached results.
    /// Example: `old.part_replay_change(&edited)` returns the dirty tail, if any.
    pub fn part_replay_change(&self, candidate: &Self) -> Option<FeatureReplayRange> {
        let previous = self.effective_work_geometry();
        let current = candidate.effective_work_geometry();
        let end = current.effective_end();
        if !self.sketches.same_evaluated_state(&candidate.sketches) {
            return Some(FeatureReplayRange { start: 0, end });
        }
        let old_end = previous.effective_end();
        let shared = old_end.min(end);
        let mismatch = previous.features()[..shared]
            .iter()
            .zip(&current.features()[..shared])
            .position(|(old, new)| !old.same_replay_recipe(new));
        let mut start = mismatch.or_else(|| (old_end != end).then_some(shared));
        if !self.parameters.same_evaluated_state(&candidate.parameters) {
            if let Some(changed) = self
                .parameters
                .changed_evaluated_parameters(&candidate.parameters)
            {
                let parameter_start =
                    changed_parameter_start(self, candidate, &changed).unwrap_or(0);
                start = Some(start.map_or(parameter_start, |index| index.min(parameter_start)));
            } else if !safe_appended_parameters(self, candidate, &previous, start, old_end, end) {
                return Some(FeatureReplayRange { start: 0, end });
            }
        }
        let start = start?;
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

fn safe_appended_parameters(
    source: &DocumentSnapshot,
    candidate: &DocumentSnapshot,
    previous: &crate::feature::FeatureSequence<super::PartFeatureDefinition>,
    start: Option<usize>,
    old_end: usize,
    end: usize,
) -> bool {
    if start != Some(old_end) || end <= old_end {
        return false;
    }
    let Some(added) = source
        .parameters
        .appended_unchanged_parameters(&candidate.parameters)
    else {
        return false;
    };
    if added.is_empty() {
        return false;
    }
    parameter_reader_start(source, previous, &added) == Some(old_end)
}

fn changed_parameter_start(
    source: &DocumentSnapshot,
    candidate: &DocumentSnapshot,
    changed: &std::collections::BTreeSet<u64>,
) -> Option<usize> {
    let old = source.effective_work_geometry();
    let new = candidate.effective_work_geometry();
    let old_start = parameter_reader_start(source, &old, changed)?;
    let new_start = parameter_reader_start(candidate, &new, changed)?;
    Some(old_start.min(new_start).min(new.effective_end()))
}
fn parameter_reader_start(
    snapshot: &DocumentSnapshot,
    sequence: &crate::feature::FeatureSequence<super::PartFeatureDefinition>,
    changed: &std::collections::BTreeSet<u64>,
) -> Option<usize> {
    for (index, feature) in sequence.features()[..sequence.effective_end()]
        .iter()
        .enumerate()
    {
        if feature_reads_changed(snapshot, feature, changed)? {
            return Some(index);
        }
    }
    Some(sequence.effective_end())
}
fn feature_reads_changed(
    snapshot: &DocumentSnapshot,
    feature: &crate::feature::AuthoredFeature<super::PartFeatureDefinition>,
    changed: &std::collections::BTreeSet<u64>,
) -> Option<bool> {
    if feature.reads().has_unattributed_reads() {
        return None;
    }
    let reads = match feature.definition() {
        super::PartFeatureDefinition::SketchPrism(definition) => definition.reads(snapshot).ok()?,
        super::PartFeatureDefinition::SketchLoft(definition) => definition.reads(snapshot).ok()?,
        super::PartFeatureDefinition::SketchRevolve(definition) => {
            definition.reads(snapshot).ok()?
        }
        super::PartFeatureDefinition::SketchCoil(definition) => definition.reads(snapshot).ok()?,
        super::PartFeatureDefinition::SketchSweep(definition) => definition.reads(snapshot).ok()?,
        definition => definition.reads(),
    };
    if reads.has_unattributed_reads() {
        return None;
    }
    Some(changed.iter().any(|id| {
        let key = FeatureReadKey::Parameter(*id);
        feature.reads().keys().contains(&key) || reads.keys().contains(&key)
    }))
}
