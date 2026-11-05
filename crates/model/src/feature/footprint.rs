// SPDX-License-Identifier: GPL-2.0-only

use super::FeatureId;
use std::collections::BTreeSet;

/// Document-local read identities, separate namespaces even when numbers match.
#[derive(
    Clone, Copy, Debug, Eq, Ord, PartialEq, PartialOrd, serde::Serialize, serde::Deserialize,
)]
pub enum FeatureReadKey {
    Feature(FeatureId),
    Parameter(u64),
    Sketch(u64),
}

/// Explicit reads and conservative attribution status; empty is not unknown.
#[derive(Clone, Debug, Default, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FeatureReadFootprint {
    pub(super) keys: BTreeSet<FeatureReadKey>,
    unattributed: bool,
}

impl FeatureReadFootprint {
    /// Deduplicate owned reads; example: `FeatureReadFootprint::new([FeatureReadKey::Parameter(1)], false)`.
    pub fn new(keys: impl IntoIterator<Item = FeatureReadKey>, unattributed: bool) -> Self {
        Self {
            keys: keys.into_iter().collect(),
            unattributed,
        }
    }

    /// Inspect deterministic attributed keys; example: `footprint.keys().contains(&key)`.
    pub fn keys(&self) -> &BTreeSet<FeatureReadKey> {
        &self.keys
    }

    /// Inspect conservative read status; example: `footprint.has_unattributed_reads()`.
    pub fn has_unattributed_reads(&self) -> bool {
        self.unattributed
    }

    pub(super) fn feature_drivers(&self) -> Vec<FeatureId> {
        self.keys
            .iter()
            .filter_map(|key| match key {
                FeatureReadKey::Feature(id) => Some(*id),
                _ => None,
            })
            .collect()
    }

    pub(super) fn affected_by(&self, changed: &BTreeSet<FeatureReadKey>) -> bool {
        !changed.is_empty() && (self.unattributed || !self.keys.is_disjoint(changed))
    }
}
