// SPDX-License-Identifier: GPL-2.0-only
//! Atomically replace source geometry with its block definition and instance.
use super::DocumentSnapshot;
use crate::{
    document_sketch::{DocumentSketchId, SketchBlockInstance},
    sketch::{SketchEntityId, SketchError},
};

impl DocumentSnapshot {
    /// Group selected geometry in place; example: `snapshot.create_sketch_block(sketch, "Rib", &[line], instance_id)?`.
    pub fn create_sketch_block(
        &mut self,
        sketch: DocumentSketchId,
        name: &str,
        selected: &[SketchEntityId],
        instance_id: SketchEntityId,
    ) -> Result<usize, SketchError> {
        if self.work_geometry.sketch_consumers(sketch).next().is_some() {
            return Err(SketchError::Definition(format!(
                "block selection {selected:?} in sketch {sketch:?}: expected a sketch without consuming features until block profiles can be replayed"
            )));
        }
        let mut candidate = self.clone();
        let index = candidate.extract_block(sketch, name, selected, instance_id)?;
        *self = candidate;
        Ok(index)
    }

    fn extract_block(
        &mut self,
        sketch: DocumentSketchId,
        name: &str,
        selected: &[SketchEntityId],
        instance_id: SketchEntityId,
    ) -> Result<usize, SketchError> {
        let placed = self.sketches.get(sketch)?;
        let sources = placed
            .geometry()
            .dimensional_constraints()
            .map(|(_, id)| id)
            .collect();
        let mut geometry = placed.geometry().clone();
        let (captured, _) = geometry.extract_block_geometry(selected)?;
        let owned = captured.owned_entity_ids();
        let bindings = placed
            .point_projections()
            .iter()
            .copied()
            .filter(|binding| !owned.contains(&binding.anchor))
            .collect();
        let index = self.sketches.define_block_geometry(name, captured)?;
        self.sketches.set_point_projections(sketch, bindings)?;
        self.sketches.replace_geometry(sketch, geometry)?;
        self.remove_retired_dimension_sources(sources)?;
        let instance = SketchBlockInstance::new(instance_id, name, [0., 0.], 0., 1.)?;
        self.sketches.insert_block_instance(sketch, instance)?;
        Ok(index)
    }
}
