// SPDX-License-Identifier: GPL-2.0-only
//! Spatial sketches share the document identity reservations with planar sketches.
use super::{DocumentSketchId, DocumentSketches, SketchError};
use crate::spatial_sketch::{SpatialSketch, SpatialSketchRecipe};

impl DocumentSketches {
    /// Inspect spatial sketches in creation order; example: `document.spatial_sketches()`.
    pub fn spatial_sketches(&self) -> &[SpatialSketch] {
        &self.spatial_sketches
    }

    /// Add validated geometry with caller-reserved identities; example: `document.add_spatial(sketch)?`.
    pub fn add_spatial(&mut self, sketch: SpatialSketch) -> Result<(), SketchError> {
        if self.sketch_ids().contains(&sketch.id().0) {
            return Err(SketchError::Definition(format!(
                "document sketch ID {:?}: expected a unique identity",
                sketch.id()
            )));
        }
        let retained = self.entity_ids();
        for id in spatial_ids(&sketch) {
            if retained.contains(&id) {
                return Err(SketchError::Definition(format!(
                    "document sketch entity ID {id}: expected one owner across all document sketches"
                )));
            }
        }
        self.spatial_sketches.push(sketch);
        Ok(())
    }

    /// Remove spatial geometry while retaining document reservations; example: `document.remove_spatial(id)?`.
    pub fn remove_spatial(&mut self, id: DocumentSketchId) -> Result<SpatialSketch, SketchError> {
        let index = self
            .spatial_sketches
            .iter()
            .position(|sketch| sketch.id() == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "spatial sketch ID {id:?}: expected an existing spatial sketch"
                ))
            })?;
        Ok(self.spatial_sketches.remove(index))
    }

    pub(super) fn spatial_entity_ids(&self) -> impl Iterator<Item = u64> + '_ {
        self.spatial_sketches.iter().flat_map(spatial_ids)
    }

    pub(super) fn restore_spatial_sketches(
        &mut self,
        recipes: &[SpatialSketchRecipe],
    ) -> Result<(), SketchError> {
        for recipe in recipes {
            let sketch = SpatialSketch::from_recipe(recipe).map_err(SketchError::Definition)?;
            self.add_spatial(sketch)?;
        }
        Ok(())
    }
}

fn spatial_ids(sketch: &SpatialSketch) -> impl Iterator<Item = u64> + '_ {
    sketch
        .points()
        .iter()
        .map(|point| point.id.0)
        .chain(sketch.lines().iter().map(|line| line.id.0))
}
