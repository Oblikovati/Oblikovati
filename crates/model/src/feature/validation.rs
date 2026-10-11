// SPDX-License-Identifier: GPL-2.0-only

use super::{AuthoredFeature, FeatureId, FeatureSequence, FeatureSequenceError};
use crate::dependency::{DependencyError, DependencyGraph};
use std::collections::BTreeMap;

impl<Definition> FeatureSequence<Definition> {
    pub(super) fn position(&self, id: FeatureId) -> Result<usize, FeatureSequenceError> {
        self.features
            .iter()
            .position(|feature| feature.id == id)
            .ok_or_else(|| DependencyError::Missing(id).into())
    }

    pub(super) fn validate_order(
        order: &[&AuthoredFeature<Definition>],
    ) -> Result<(), FeatureSequenceError> {
        let positions: BTreeMap<FeatureId, usize> = order
            .iter()
            .enumerate()
            .map(|(index, feature)| (feature.id, index))
            .collect();
        for (index, feature) in order.iter().enumerate() {
            Self::validate_driver_order(feature, index, &positions)?;
        }
        Ok(())
    }

    fn validate_driver_order(
        feature: &AuthoredFeature<Definition>,
        index: usize,
        positions: &BTreeMap<FeatureId, usize>,
    ) -> Result<(), FeatureSequenceError> {
        for driver in feature.reads.feature_drivers() {
            let position = positions
                .get(&driver)
                .ok_or(DependencyError::Missing(driver))?;
            if *position >= index {
                return Err(FeatureSequenceError::ForwardRead {
                    feature: feature.id,
                    driver,
                });
            }
        }
        Ok(())
    }

    pub(super) fn graph_with(
        &self,
        feature: &AuthoredFeature<Definition>,
        added: bool,
    ) -> Result<DependencyGraph<FeatureId>, FeatureSequenceError> {
        let mut dependencies = self.dependencies.clone();
        if added {
            dependencies.insert(feature.id)?;
        }
        dependencies.replace_drivers(feature.id, &feature.reads.feature_drivers())?;
        Ok(dependencies)
    }
}
