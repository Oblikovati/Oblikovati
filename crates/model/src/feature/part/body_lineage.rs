// SPDX-License-Identifier: GPL-2.0-only
//! Route producer namespaces to current body slots across replacing operations.
use crate::feature::FeatureId;
use std::{
    collections::{BTreeMap, BTreeSet},
    ops::RangeInclusive,
};

/// An authored output identity within its producing feature.
#[derive(Clone, Copy, Debug, Eq, Ord, PartialEq, PartialOrd)]
pub(super) struct BodySlot {
    pub(super) feature: FeatureId,
    pub(super) output: u32,
}
impl From<FeatureId> for BodySlot {
    fn from(feature: FeatureId) -> Self {
        Self { feature, output: 0 }
    }
}
impl BodySlot {
    pub(super) fn feature_range(feature: FeatureId) -> RangeInclusive<Self> {
        Self::from(feature)..=Self {
            feature,
            output: u32::MAX,
        }
    }
}

#[derive(Clone)]
struct ProducerOutputs {
    slots: BTreeSet<BodySlot>,
    grouped: bool,
}

#[derive(Clone, Default)]
pub(super) struct BodyLineageIndex {
    owners: BTreeMap<FeatureId, ProducerOutputs>,
}
impl BodyLineageIndex {
    #[cfg(test)]
    pub(super) fn insert_test_owners(&mut self, source: FeatureId, owners: &[FeatureId]) {
        self.owners.insert(
            source,
            ProducerOutputs {
                slots: owners.iter().copied().map(BodySlot::from).collect(),
                grouped: false,
            },
        );
    }
    pub(super) fn register(&mut self, feature: FeatureId) -> Result<(), String> {
        self.register_outputs(feature, BTreeSet::from([BodySlot::from(feature)]))
    }
    pub(super) fn register_outputs(
        &mut self,
        feature: FeatureId,
        slots: BTreeSet<BodySlot>,
    ) -> Result<(), String> {
        if self.owners.contains_key(&feature)
            || slots.is_empty()
            || slots.iter().any(|slot| slot.feature != feature)
        {
            return Err(format!(
                "body feature {feature:?}, slots {slots:?}: expected an unused producer and its own output slots"
            ));
        }
        self.owners.insert(
            feature,
            ProducerOutputs {
                grouped: slots.len() > 1,
                slots,
            },
        );
        Ok(())
    }
    pub(super) fn has_grouped_outputs(&self, source: FeatureId) -> bool {
        self.owners
            .get(&source)
            .is_some_and(|outputs| outputs.grouped)
    }
    pub(super) fn candidates(&self, source: FeatureId) -> impl Iterator<Item = BodySlot> + '_ {
        self.owners
            .get(&source)
            .into_iter()
            .flat_map(|owners| owners.slots.iter().copied())
    }
    pub(super) fn replace(&mut self, current: BodySlot, next: FeatureId) -> Result<(), String> {
        if self.owners.contains_key(&next)
            || !self
                .owners
                .values()
                .any(|owners| owners.slots.contains(&current))
        {
            return Err(format!(
                "body replacement {current:?} -> {next:?}: expected a live source and unused target"
            ));
        }
        self.register(next)?;
        for owners in self.owners.values_mut() {
            if owners.slots.remove(&current) {
                owners.slots.insert(BodySlot::from(next));
            }
        }
        Ok(())
    }
    pub(super) fn retire(&mut self, current: BodySlot) {
        self.owners.retain(|_, owners| {
            owners.slots.remove(&current);
            !owners.slots.is_empty()
        });
    }
}
