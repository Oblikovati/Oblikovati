// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityDeletion, SketchEntityId, SketchError};
use crate::sketch::SketchConstraintId;
use std::collections::BTreeSet;

/// Typed editor intent; relation categories are checked against the seed, before any removal.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct SketchDeletionSelection {
    pub entities: Vec<SketchEntityId>,
    pub dimensions: Vec<SketchConstraintId>,
    pub geometric_constraints: Vec<SketchConstraintId>,
}
impl Sketch2D {
    /// Remove one mixed selection atomically, retiring each relation once in authored order.
    /// Example: `let retired = sketch.delete_selection(&selection)?;`.
    pub fn delete_selection(
        &mut self,
        selected: &SketchDeletionSelection,
    ) -> Result<SketchEntityDeletion, SketchError> {
        let relations = self.validate_selected_relations(selected)?;
        let mut candidate = self.clone();
        for id in &relations {
            let dimensional = selected.dimensions.contains(id);
            candidate.remove_authored_relation(*id, dimensional)?;
        }
        let mut report = candidate.delete_entities(&selected.entities)?;
        let mut retired = relations;
        retired.extend(report.constraints.iter().map(|row| row.id));
        report.constraints = self
            .constraints
            .iter()
            .filter(|row| retired.contains(&row.id))
            .cloned()
            .collect();
        *self = candidate;
        Ok(report)
    }
    fn validate_selected_relations(
        &self,
        selected: &SketchDeletionSelection,
    ) -> Result<BTreeSet<SketchConstraintId>, SketchError> {
        for id in &selected.entities {
            self.entity_kind(*id)?;
        }
        for id in &selected.dimensions {
            self.dimension_source(*id)?;
        }
        for id in &selected.geometric_constraints {
            self.geometric_constraint_index(*id)?;
        }
        Ok(selected
            .dimensions
            .iter()
            .chain(&selected.geometric_constraints)
            .copied()
            .collect())
    }
}
