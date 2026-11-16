// SPDX-License-Identifier: GPL-2.0-only
//! Route producer namespaces to current body slots across replacing operations.
use crate::feature::FeatureId;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Default)]
pub(super) struct BodyLineageIndex {
    owners: BTreeMap<FeatureId, BTreeSet<FeatureId>>,
}
impl BodyLineageIndex {
    #[cfg(test)]
    pub(super) fn insert_test_owners(&mut self, source: FeatureId, owners: &[FeatureId]) {
        self.owners.insert(source, owners.iter().copied().collect());
    }
    pub(super) fn register(&mut self, feature: FeatureId) -> Result<(), String> {
        if self.owners.contains_key(&feature) {
            return Err(format!(
                "body feature {feature:?}: expected a new producer namespace"
            ));
        }
        self.owners.insert(feature, BTreeSet::from([feature]));
        Ok(())
    }
    pub(super) fn candidates(&self, source: FeatureId) -> impl Iterator<Item = FeatureId> + '_ {
        self.owners
            .get(&source)
            .into_iter()
            .flat_map(|owners| owners.iter().copied())
    }
    pub(super) fn replace(&mut self, current: FeatureId, next: FeatureId) -> Result<(), String> {
        if self.owners.contains_key(&next)
            || !self.owners.values().any(|owners| owners.contains(&current))
        {
            return Err(format!(
                "body replacement {current:?} -> {next:?}: expected a live source and unused target"
            ));
        }
        for owners in self.owners.values_mut() {
            if owners.remove(&current) {
                owners.insert(next);
            }
        }
        self.owners.insert(next, BTreeSet::from([next]));
        Ok(())
    }
    pub(super) fn retire(&mut self, current: FeatureId) {
        self.owners.retain(|_, owners| {
            owners.remove(&current);
            !owners.is_empty()
        });
    }
}
