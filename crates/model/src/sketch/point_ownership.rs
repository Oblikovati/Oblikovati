// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};
use std::collections::BTreeSet;
impl Sketch2D {
    pub(crate) fn expose_defining_point(&mut self, id: SketchEntityId) -> Result<(), SketchError> {
        self.point(id)?;
        if !self.defining_points.remove(&id) {
            return Err(SketchError::Definition(format!(
                "placement point {id:?}: expected a hidden defining point, not an existing visible member"
            )));
        }
        self.entity_order.push(id);
        Ok(())
    }

    /// Adopt a shared solver point without a standalone marker; example: `sketch.add_defining_point(id, [0.0, 0.0])?`.
    pub fn add_defining_point(
        &mut self,
        id: SketchEntityId,
        position: [f64; 2],
    ) -> Result<(), SketchError> {
        self.add_point(id, position)?;
        self.defining_points.insert(id);
        self.entity_order.pop();
        Ok(())
    }
    /// Include hidden defining-point IDs in document reservations; example: `sketch.owned_entity_ids()`.
    pub fn owned_entity_ids(&self) -> Vec<SketchEntityId> {
        let mut ids = self.entity_order.clone();
        ids.extend(
            self.points
                .iter()
                .filter(|point| self.defining_points.contains(&point.id))
                .map(|point| point.id),
        );
        ids
    }
    pub(super) fn capture_point_ownership_schema(&self, recipe: &mut SketchRecipe) {
        if self.defining_points.is_empty() {
            return;
        }
        recipe.version = 12;
        self.capture_complete_schema_metadata(recipe);
        recipe.splines = Some(self.splines.clone());
        recipe.offset_splines = Some(self.offset_splines.clone());
        recipe.defining_points = Some(
            self.points
                .iter()
                .filter(|point| self.defining_points.contains(&point.id))
                .map(|point| point.id)
                .collect(),
        );
    }
    pub(super) fn restore_point_ownership(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        let Some(ids) = &recipe.defining_points else {
            return Ok(());
        };
        let selected: BTreeSet<SketchEntityId> = ids.iter().copied().collect();
        let expected: Vec<SketchEntityId> = self
            .points
            .iter()
            .filter(|point| selected.contains(&point.id))
            .map(|point| point.id)
            .collect();
        if *ids != expected {
            return Err(SketchError::Definition(format!(
                "defining points {ids:?}: expected distinct owned point IDs in authored order {expected:?}"
            )));
        }
        self.entity_order.retain(|id| !selected.contains(id));
        self.defining_points = selected;
        Ok(())
    }
}
pub(super) fn require_point_ownership_schema(recipe: &SketchRecipe) -> Result<(), SketchError> {
    match (recipe.version, &recipe.defining_points) {
        (1..=11, None) | (12, Some(_)) => Ok(()),
        shape => Err(SketchError::Definition(format!(
            "point ownership recipe {shape:?}: expected no defining-point field in versions 1-11 or an explicit list in version 12"
        ))),
    }
}
