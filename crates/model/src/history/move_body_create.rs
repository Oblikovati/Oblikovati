// SPDX-License-Identifier: GPL-2.0-only
//! Create an authored body motion in part history (#4097).
use super::*;
use crate::feature::{FeatureId, move_body::MoveBodyDefinition, part::PartFeatureDefinition};
impl DocumentEditHistory {
    /// Add a motion in a transaction; example: `history.create_move_body(&token, "Move", definition)?`.
    pub fn create_move_body(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: MoveBodyDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            snapshot.create_move_body(name, definition, ids)
        })
    }
}
impl DocumentSnapshot {
    /// Append authored motion inputs; example: `snapshot.create_move_body("Move", definition, ids)?`.
    pub fn create_move_body(
        &mut self,
        name: &str,
        definition: MoveBodyDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::MoveBody(Box::new(definition)),
            reads,
            ids,
        )
    }
}
