// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};
use std::collections::BTreeSet;
impl Sketch2D {
    /// Read an owned entity's role; plain points have no construction flag.
    /// Example: `sketch.is_construction(line)?`.
    pub fn is_construction(&self, entity: SketchEntityId) -> Result<bool, SketchError> {
        if !self.entity_order.contains(&entity) {
            return Err(SketchError::Definition(format!(
                "sketch entity {entity:?}: expected an owned entity for construction inspection"
            )));
        }
        Ok(self.construction_curves.contains(&entity) || self.centerlines.contains(&entity))
    }
    /// Set one checked curve selection atomically; return changed IDs in authored order.
    /// Example: `let changed = sketch.set_curve_construction(&[line, arc], true)?;`.
    pub fn set_curve_construction(
        &mut self,
        entities: &[SketchEntityId],
        construction: bool,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let selected = self.validate_curve_selection(entities)?;
        let changed: Vec<SketchEntityId> = self
            .entity_order
            .iter()
            .copied()
            .filter(|id| {
                selected.contains(id) && self.construction_curves.contains(id) != construction
            })
            .collect();
        for id in &changed {
            if construction {
                self.construction_curves.insert(*id);
            } else {
                self.construction_curves.remove(id);
            }
        }
        Ok(changed)
    }
    /// Select profile curves in authored order, excluding construction but keeping shared points.
    /// Example: `let boundary_candidates = sketch.profile_curve_ids();`.
    pub fn profile_curve_ids(&self) -> Vec<SketchEntityId> {
        let curves = self.curve_identity_set();
        self.entity_order
            .iter()
            .copied()
            .filter(|id| {
                curves.contains(id)
                    && !self.construction_curves.contains(id)
                    && !self.centerlines.contains(id)
            })
            .collect()
    }
    fn curve_identity_set(&self) -> BTreeSet<SketchEntityId> {
        self.lines
            .iter()
            .map(|shape| shape.id)
            .chain(self.circles.iter().map(|shape| shape.id))
            .chain(self.arcs.iter().map(|shape| shape.id))
            .chain(self.ellipses.iter().map(|shape| shape.id()))
            .chain(self.splines.iter().map(|shape| shape.id()))
            .chain(self.offset_splines.iter().map(|shape| shape.id()))
            .collect()
    }
    fn validate_curve_selection(
        &self,
        entities: &[SketchEntityId],
    ) -> Result<BTreeSet<SketchEntityId>, SketchError> {
        let selected: BTreeSet<SketchEntityId> = entities.iter().copied().collect();
        let owned = self.curve_identity_set();
        if selected.len() != entities.len() || !selected.is_subset(&owned) {
            return Err(SketchError::Definition(format!(
                "construction selection {entities:?}: expected unique owned line/circle/arc/ellipse IDs from {owned:?}"
            )));
        }
        Ok(selected)
    }
    pub(super) fn saved_construction_curves(&self) -> Option<Vec<SketchEntityId>> {
        if self.construction_curves.is_empty() && self.centerlines.is_empty() {
            return None;
        }
        Some(
            self.entity_order
                .iter()
                .copied()
                .filter(|id| self.construction_curves.contains(id))
                .collect(),
        )
    }
    pub(super) fn restore_construction_curves(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.construction_curves) {
            (1 | 2, None) => Ok(()),
            (3..=10, Some(curves)) => self.adopt_construction_curves(curves),
            shape => Err(SketchError::Definition(format!(
                "construction recipe {shape:?}: expected no metadata in versions 1/2 or explicit curve metadata in versions 3-10"
            ))),
        }
    }
    fn adopt_construction_curves(&mut self, curves: &[SketchEntityId]) -> Result<(), SketchError> {
        let selected = self.validate_curve_selection(curves)?;
        let expected: Vec<SketchEntityId> = self
            .entity_order
            .iter()
            .copied()
            .filter(|id| selected.contains(id))
            .collect();
        if curves != expected {
            return Err(SketchError::Definition(format!(
                "construction curves {curves:?}: expected authored identity order {expected:?}"
            )));
        }
        self.construction_curves = selected;
        Ok(())
    }
}
