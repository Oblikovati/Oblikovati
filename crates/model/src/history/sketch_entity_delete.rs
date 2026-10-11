// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityDeletion, SketchEntityId, SketchError},
};
use oblikovati_api::types::MODEL_PARAM;
use std::collections::BTreeSet;

impl DocumentSnapshot {
    /// Remove geometry and its unshared Model dimensions atomically, retaining unrelated sources.
    /// Example: `snapshot.delete_sketch_entities(sketch, &[line])?;`.
    pub fn delete_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
    ) -> Result<SketchEntityDeletion, SketchError> {
        let placed = self.sketches.get(sketch)?;
        let sources = dimension_sources(placed.geometry());
        let mut geometry = placed.geometry().clone();
        let report = geometry.delete_entities(entities)?;
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

    fn remove_retired_dimension_sources(
        &mut self,
        sources: BTreeSet<u64>,
    ) -> Result<(), SketchError> {
        for parameter in sources {
            if self.retains_dimension_source(parameter) || !self.is_model_source(parameter) {
                continue;
            }
            self.parameters.validate_owned_scalar(parameter)
                .and_then(|()| self.parameters.remove_for_owner(parameter))
                .map_err(|cause| SketchError::Definition(format!(
                    "deleted dimension parameter {parameter}: expected a removable Model scalar: {cause}"
                )))?;
        }
        Ok(())
    }

    fn is_model_source(&self, parameter: u64) -> bool {
        self.parameters
            .get(parameter)
            .is_some_and(|source| source.kind == MODEL_PARAM)
    }

    fn retains_dimension_source(&self, parameter: u64) -> bool {
        self.sketches.sketches().iter().any(|placed| {
            placed
                .geometry()
                .dimensional_constraints()
                .any(|(_, source)| source == parameter)
        }) || self.work_geometry.features().iter().any(|feature| {
            feature
                .owned_scalars()
                .iter()
                .any(|binding| binding.parameter == parameter)
        })
    }
}

fn dimension_sources(geometry: &crate::sketch::Sketch2D) -> BTreeSet<u64> {
    geometry
        .dimensional_constraints()
        .map(|(_, parameter)| parameter)
        .collect()
}
