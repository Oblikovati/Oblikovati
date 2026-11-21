// SPDX-License-Identifier: GPL-2.0-only
//! Move selected geometry into a separate sketch without damaging shared source points.
use super::{
    Sketch2D, SketchDeletionSelection, SketchEntityDeletion, SketchEntityId, SketchError,
    circular_compile, dependencies, entity_geometry, residual::PlanarEquation,
};
use std::collections::BTreeSet;

impl Sketch2D {
    /// Extract geometry and its touching relations atomically; example: `sketch.extract_block_geometry(&[line])?`.
    pub fn extract_block_geometry(
        &mut self,
        selected: &[SketchEntityId],
    ) -> Result<(Sketch2D, SketchEntityDeletion), SketchError> {
        let captured = self.capture_selection(selected)?;
        let selection = self.block_deletion_selection(selected)?;
        let mut candidate = self.clone();
        let removable = candidate.hide_shared_selected_points(selected, &selection.entities);
        let mut deletion = selection;
        deletion.entities = removable;
        let report = candidate.delete_selection(&deletion)?;
        self.check_block_dependents(selected, &report)?;
        *self = candidate;
        Ok((captured, report))
    }

    fn block_deletion_selection(
        &self,
        selected: &[SketchEntityId],
    ) -> Result<SketchDeletionSelection, SketchError> {
        let catalogue = entity_geometry::catalogue(self);
        let mut columns = BTreeSet::new();
        for id in selected {
            columns.extend(catalogue[id].columns(self)?);
        }
        let mut selection = SketchDeletionSelection {
            entities: selected.to_vec(),
            ..Default::default()
        };
        for constraint in &self.constraints {
            let prepared = circular_compile::prepare_relation(
                self,
                &constraint.relation,
                PlanarEquation::Coincident,
            )?;
            if prepared
                .variables
                .iter()
                .any(|column| columns.contains(column))
            {
                let ids = if dependencies::dimension_parameter(&constraint.relation).is_some() {
                    &mut selection.dimensions
                } else {
                    &mut selection.geometric_constraints
                };
                ids.push(constraint.id);
            }
        }
        Ok(selection)
    }

    fn hide_shared_selected_points(
        &mut self,
        selected: &[SketchEntityId],
        entities: &[SketchEntityId],
    ) -> Vec<SketchEntityId> {
        let selected: BTreeSet<_> = selected.iter().copied().collect();
        let shared: BTreeSet<_> = entity_geometry::catalogue(self)
            .into_iter()
            .filter(|(id, _)| !selected.contains(id))
            .flat_map(|(_, shape)| shape.ownership_dependencies())
            .collect();
        let hidden: BTreeSet<_> = entities
            .iter()
            .copied()
            .filter(|id| shared.contains(id) && self.point(*id).is_ok())
            .collect();
        self.entity_order.retain(|id| !hidden.contains(id));
        self.defining_points.extend(hidden.iter().copied());
        entities
            .iter()
            .copied()
            .filter(|id| !hidden.contains(id))
            .collect()
    }

    fn check_block_dependents(
        &self,
        selected: &[SketchEntityId],
        report: &SketchEntityDeletion,
    ) -> Result<(), SketchError> {
        let visible = self.entity_ids();
        let unexpected: Vec<_> = report
            .entities
            .iter()
            .copied()
            .filter(|id| visible.contains(id) && !selected.contains(id))
            .collect();
        if !unexpected.is_empty() {
            return Err(SketchError::Definition(format!(
                "block selection {selected:?} would remove dependent entities {unexpected:?}: expected all dependent curves in the selection"
            )));
        }
        Ok(())
    }
}
