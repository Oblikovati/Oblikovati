// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchDeletionSelection, SketchEntityDeletion, SketchEntityId, SketchError},
};
use std::collections::BTreeSet;

impl DocumentSnapshot {
    /// Remove geometry and its unshared Model dimensions atomically, retaining unrelated sources.
    /// Example: `snapshot.delete_sketch_entities(sketch, &[line])?;`.
    pub fn delete_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
    ) -> Result<SketchEntityDeletion, SketchError> {
        let selection = SketchDeletionSelection {
            entities: entities.to_vec(),
            ..Default::default()
        };
        self.delete_sketch_selection(sketch, &selection)
    }
    /// Apply one typed editor selection with a single document source and projection cascade.
    /// Example: `snapshot.delete_sketch_selection(sketch, &selection)?;`.
    pub fn delete_sketch_selection(
        &mut self,
        sketch: DocumentSketchId,
        selection: &SketchDeletionSelection,
    ) -> Result<SketchEntityDeletion, SketchError> {
        let placed = self.sketches.get(sketch)?;
        let sources = dimension_sources(placed.geometry());
        let mut geometry = placed.geometry().clone();
        let report = geometry.delete_selection(selection)?;
        let bindings = placed
            .point_projections()
            .iter()
            .copied()
            .filter(|binding| !report.entities.contains(&binding.anchor))
            .collect();
        let mut candidate = self.clone();
        candidate.sketches.set_point_projections(sketch, bindings)?;
        candidate.sketches.replace_geometry(sketch, geometry)?;
        candidate.remove_retired_dimension_sources(sources)?;
        *self = candidate;
        Ok(report)
    }
}

fn dimension_sources(geometry: &crate::sketch::Sketch2D) -> BTreeSet<u64> {
    geometry
        .dimensional_constraints()
        .map(|(_, parameter)| parameter)
        .collect()
}
