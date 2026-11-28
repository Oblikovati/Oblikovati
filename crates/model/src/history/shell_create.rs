// SPDX-License-Identifier: GPL-2.0-only
//! Create an authored Shell in the common part history (#4106).
use super::*;
use crate::feature::{FeatureId, part::PartFeatureDefinition, shell::ShellDefinition};

impl DocumentEditHistory {
    /// Add a Shell inside a transaction; example: `history.create_shell(&token, "Shell", definition)?`.
    pub fn create_shell(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: ShellDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            snapshot.create_shell(name, definition, ids)
        })
    }
}
impl DocumentSnapshot {
    /// Append Shell inputs; example: `snapshot.create_shell("Shell", definition, ids)?`.
    pub fn create_shell(
        &mut self,
        name: &str,
        definition: ShellDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Shell(Box::new(definition)),
            reads,
            ids,
        )
    }
    /// Replace an existing Shell definition in-place; example: `snapshot.edit_shell(id, definition)?`.
    pub fn edit_shell(
        &mut self,
        id: FeatureId,
        definition: ShellDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                id,
                PartFeatureDefinition::Shell(Box::new(definition)),
                reads,
            )
            .map(|_| ())
            .map_err(|cause| {
                HistoryError::PartProgram(
                    crate::feature::part::PartFeatureProgramError::Sequence(cause),
                )
            })
    }
}
