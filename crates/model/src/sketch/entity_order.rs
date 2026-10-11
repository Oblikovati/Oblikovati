// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};
use std::collections::BTreeSet;
impl Sketch2D {
    /// Inspect entity identities in insertion order across all geometry kinds.
    /// Example: `sketch.entity_ids()` for a stable API enumeration.
    pub fn entity_ids(&self) -> &[SketchEntityId] {
        &self.entity_order
    }
    /// Classify an owned entity without a solve or derived movement analysis.
    /// Example: `let kind = sketch.entity_kind(selected_id)?;` before preparing a constructor.
    pub fn entity_kind(
        &self,
        id: SketchEntityId,
    ) -> Result<oblikovati_api::types::SketchEntityKind, SketchError> {
        super::entity_geometry::catalogue(self)
            .get(&id)
            .map(|shape| shape.kind())
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch entity {id:?}: expected one owned point, line, circle, or arc"
                ))
            })
    }
    pub(super) fn saved_entity_order(&self) -> Option<Vec<SketchEntityId>> {
        (self.entity_order != self.grouped_entity_order()
            || !self.construction_curves.is_empty()
            || !self.centerlines.is_empty())
        .then(|| self.entity_order.clone())
    }
    fn grouped_entity_order(&self) -> Vec<SketchEntityId> {
        self.points
            .iter()
            .map(|point| point.id)
            .chain(self.lines.iter().map(|line| line.id))
            .chain(self.circles.iter().map(|circle| circle.id))
            .chain(self.arcs.iter().map(|arc| arc.id))
            .collect()
    }
    pub(super) fn restore_entity_order(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.entity_order) {
            (1, None) => Ok(()),
            (2..=5, Some(order)) => self.adopt_entity_order(order),
            shape => Err(SketchError::Definition(format!(
                "sketch entity order {shape:?}: expected no order in version 1 or an exact permutation in version 2, version 3, version 4, or version 5"
            ))),
        }
    }
    fn adopt_entity_order(&mut self, order: &[SketchEntityId]) -> Result<(), SketchError> {
        let expected: BTreeSet<SketchEntityId> = self.entity_order.iter().copied().collect();
        let actual: BTreeSet<SketchEntityId> = order.iter().copied().collect();
        if order.len() != self.entity_order.len() || actual != expected {
            return Err(SketchError::Definition(format!(
                "sketch entity order {order:?}: expected each owned identity exactly once from {expected:?}"
            )));
        }
        self.entity_order = order.to_vec();
        Ok(())
    }
}
