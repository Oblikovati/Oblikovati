// SPDX-License-Identifier: GPL-2.0-only
use super::WorkGeometryDefinition;
use crate::feature::{AuthoredFeature, FeatureId, FeatureSequence, FeatureSequenceError};

impl AuthoredFeature<WorkGeometryDefinition> {
    /// Inspect viewport visibility independently of construction state; example: `datum.is_visible()`.
    pub fn is_visible(&self) -> bool {
        self.datum_visible
    }
}
impl FeatureSequence<WorkGeometryDefinition> {
    /// Change only authored visibility; example: `datums.set_datum_visible(id, false)?`.
    pub fn set_datum_visible(
        &mut self,
        id: FeatureId,
        visible: bool,
    ) -> Result<(), FeatureSequenceError> {
        let position = self.position(id)?;
        self.features[position].datum_visible = visible;
        Ok(())
    }
}
