// SPDX-License-Identifier: GPL-2.0-only
//! Create an authored Hole in the common part history (#4093).
use super::*;
use crate::feature::{FeatureId, hole::HoleDefinition, part::PartFeatureDefinition};

impl DocumentEditHistory {
    /// Add a Hole inside a transaction; example: `history.create_hole(&token, "Hole", definition)?`.
    pub fn create_hole(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: HoleDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            snapshot.create_hole(name, definition, ids)
        })
    }
}
impl DocumentSnapshot {
    /// Append Hole inputs; example: `snapshot.create_hole("Hole", definition, ids)?`.
    pub fn create_hole(
        &mut self,
        name: &str,
        definition: HoleDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Hole(Box::new(definition)),
            reads,
            ids,
        )
    }
}
