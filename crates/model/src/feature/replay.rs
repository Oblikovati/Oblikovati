// SPDX-License-Identifier: GPL-2.0-only

use super::{FeatureReadKey, FeatureSequence, FeatureSequenceError};
use std::collections::BTreeSet;

/// Reuse state before start, discard the old tail, and replay [start,end).
/// An empty range still truncates output after removal or end-of-part movement.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct FeatureReplayRange {
    pub start: usize,
    pub end: usize,
}

impl<Definition> FeatureSequence<Definition> {
    /// Find the first active affected reader; example: `sequence.replay_for_reads([FeatureReadKey::Parameter(1)])`.
    pub fn replay_for_reads(
        &self,
        changed: impl IntoIterator<Item = FeatureReadKey>,
    ) -> Option<FeatureReplayRange> {
        let changed: BTreeSet<FeatureReadKey> = changed.into_iter().collect();
        self.features[..self.effective_end()]
            .iter()
            .position(|feature| {
                changed.contains(&FeatureReadKey::Feature(feature.id))
                    || feature.reads.affected_by(&changed)
            })
            .map(|start| self.replay_from(start))
    }

    /// Move the exclusive cutoff; example: `sequence.set_end_of_part(Some(0))?` rolls back before the first feature.
    pub fn set_end_of_part(
        &mut self,
        cutoff: Option<usize>,
    ) -> Result<Option<FeatureReplayRange>, FeatureSequenceError> {
        if let Some(index) = cutoff.filter(|index| *index > self.features.len()) {
            return Err(FeatureSequenceError::Boundary {
                index,
                length: self.features.len(),
            });
        }
        if self.end_of_part == cutoff {
            return Ok(None);
        }
        self.end_of_part = cutoff;
        Ok(Some(self.replay_from(0)))
    }

    pub(super) fn replay_from(&self, index: usize) -> FeatureReplayRange {
        let end = self.effective_end();
        FeatureReplayRange {
            start: index.min(end),
            end,
        }
    }
}
