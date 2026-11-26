// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{
    FeatureId, circular_pattern::CircularPatternDefinition, part::PartFeatureDefinition,
    rectangular_pattern::RectangularPatternDefinition,
};

impl DocumentSnapshot {
    /// Author a feature grid in a private document candidate.
    /// Example: `snapshot.create_rectangular_feature_pattern("Pattern1", definition, ids)?`.
    pub fn create_rectangular_feature_pattern(
        &mut self,
        name: &str,
        definition: RectangularPatternDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::RectangularPattern(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Author a circular pattern in a private candidate.
    /// Example: `snapshot.create_circular_feature_pattern("Pattern1", definition, ids)?`.
    pub fn create_circular_feature_pattern(
        &mut self,
        name: &str,
        definition: CircularPatternDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::CircularPattern(Box::new(definition)),
            reads,
            ids,
        )
    }
}
