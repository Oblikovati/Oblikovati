// SPDX-License-Identifier: GPL-2.0-only

use super::{
    AuthoredFeature, FeatureId, FeatureReplayRange, FeatureSequence, FeatureSequenceError,
};

impl<Definition> FeatureSequence<Definition> {
    /// Append after validated drivers; example: `sequence.append(AuthoredFeature::new(id, "Plane", plane, reads))?`.
    pub fn append(
        &mut self,
        feature: AuthoredFeature<Definition>,
    ) -> Result<FeatureReplayRange, FeatureSequenceError> {
        let dependencies = self.graph_with(&feature, true)?;
        let start = self.features.len();
        self.dependencies = dependencies;
        self.features.push(feature);
        Ok(self.replay_from(start))
    }

    /// Remove only an unused identity, preserving the dirty tail; example: `sequence.remove(id)?`.
    pub fn remove(&mut self, id: FeatureId) -> Result<FeatureReplayRange, FeatureSequenceError> {
        let start = self.position(id)?;
        self.dependencies.remove(id).map_err(|mut error| {
            // Stable ID order keeps refusal diagnostics unchanged after recipe restore (#3728).
            if let crate::dependency::DependencyError::InUse { readers, .. } = &mut error {
                readers.sort_unstable();
            }
            FeatureSequenceError::Dependency(error)
        })?;
        self.features.remove(start);
        Ok(self.replay_from(start))
    }

    /// Move within a dependency-valid order; example: `sequence.reorder(id, 0)?`.
    pub fn reorder(
        &mut self,
        id: FeatureId,
        index: usize,
    ) -> Result<Option<FeatureReplayRange>, FeatureSequenceError> {
        let old = self.position(id)?;
        self.require_index(index)?;
        if old == index {
            return Ok(None);
        }
        let mut order: Vec<&AuthoredFeature<Definition>> = self.features.iter().collect();
        let feature = order.remove(old);
        order.insert(index, feature);
        Self::validate_order(&order)?;
        let feature = self.features.remove(old);
        self.features.insert(index, feature);
        Ok(Some(self.replay_from(old.min(index))))
    }

    /// Edit only the display label, leaving identity and replay inputs intact; example: `sequence.rename(id, "Base")?`.
    pub fn rename(&mut self, id: FeatureId, name: &str) -> Result<(), FeatureSequenceError> {
        let index = self.position(id)?;
        self.features[index].name = name.into();
        Ok(())
    }

    /// Toggle pass-through and invalidate the running body tail; example: `sequence.set_suppressed(id, true)?`.
    pub fn set_suppressed(
        &mut self,
        id: FeatureId,
        suppressed: bool,
    ) -> Result<Option<FeatureReplayRange>, FeatureSequenceError> {
        let index = self.position(id)?;
        if self.features[index].suppressed == suppressed {
            return Ok(None);
        }
        self.features[index].suppressed = suppressed;
        Ok(Some(self.replay_from(index)))
    }

    fn require_index(&self, index: usize) -> Result<(), FeatureSequenceError> {
        if index >= self.features.len() {
            return Err(FeatureSequenceError::Index {
                index,
                length: self.features.len(),
            });
        }
        Ok(())
    }
}

impl<Definition: Clone> FeatureSequence<Definition> {
    /// Bind a declared scalar read to its feature; example: `sequence.assign_scalar_owner(id, "offset", parameter)?`.
    pub fn assign_scalar_owner(
        &mut self,
        id: FeatureId,
        role: &str,
        parameter: u64,
    ) -> Result<FeatureReplayRange, FeatureSequenceError> {
        let index = self.position(id)?;
        let replacement = self.features[index]
            .clone()
            .with_owned_scalar(role, parameter)
            .map_err(FeatureSequenceError::ScalarOwnership)?;
        self.install_replacement(index, replacement)
    }
}

impl<Definition> FeatureSequence<Definition> {
    /// Replace a typed definition and reads atomically; example: `sequence.replace(id, revised_plane, reads)?`.
    pub fn replace(
        &mut self,
        id: FeatureId,
        definition: Definition,
        reads: super::FeatureReadFootprint,
    ) -> Result<FeatureReplayRange, FeatureSequenceError> {
        let index = self.position(id)?;
        let previous = &self.features[index];
        // Definitions own exact authored equality; generic PartialEq can lose signed-zero edits.
        let mut replacement = AuthoredFeature::new(id, &previous.name, definition, reads);
        replacement.suppressed = previous.suppressed;
        replacement.construction_datum = previous.construction_datum;
        replacement.datum_visible = previous.datum_visible;
        replacement.plane_normal_flipped = previous.plane_normal_flipped;
        replacement.owned_scalars = previous.owned_scalars.clone();
        replacement.store_condition(previous.condition);
        self.install_replacement(index, replacement)
    }

    fn install_replacement(
        &mut self,
        index: usize,
        replacement: AuthoredFeature<Definition>,
    ) -> Result<FeatureReplayRange, FeatureSequenceError> {
        let dependencies = self.graph_with(&replacement, false)?;
        let mut order: Vec<&AuthoredFeature<Definition>> = self.features.iter().collect();
        order[index] = &replacement;
        Self::validate_order(&order)?;
        self.features[index] = replacement;
        self.dependencies = dependencies;
        Ok(self.replay_from(index))
    }
}
