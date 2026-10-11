// SPDX-License-Identifier: GPL-2.0-only
//! Placed sketches owned by one document; allocation is a separate permanent owner.
use crate::sketch::{
    Sketch2D, SketchError, SketchPlane, SketchPlaneOptions, SketchPlaneRecipe, SketchRecipe,
};
use std::collections::BTreeSet;
mod health;
mod orientation;
pub(crate) use orientation::UnavailableSketchAxes;
pub use orientation::{SketchAxisReference, SketchAxisResolver, SketchOrientationDefinition};
mod placement;
pub use placement::SketchPlacement;
mod readiness;
mod recompute;
pub use health::{PlacedSketchHealth, SketchEvaluation, SketchSolveDiagnostics};
pub use readiness::{SketchReadinessCause, SketchReadinessError};
pub use recompute::{
    DocumentSketchRecompute, DocumentSketchSolveOutcome, DocumentSketchSolveReport,
};
/// Saved sketch identity, in a namespace distinct from entity and constraint IDs.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct DocumentSketchId(pub u64);
/// A placed planar sketch with private edit boundaries.
#[derive(Clone, Debug)]
pub struct PlacedSketch {
    id: DocumentSketchId,
    name: String,
    placement: SketchPlacement,
    orientation: Option<SketchOrientationDefinition>,
    geometry: Sketch2D,
    evaluation: Option<health::SketchEvaluationStamp>,
}
/// Ordered document state; permanent ID allocators must remain outside this snapshot.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct DocumentSketches {
    sketches: Vec<PlacedSketch>,
}
/// One authored placed sketch, without solver or normal caches.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PlacedSketchRecipe {
    pub id: DocumentSketchId,
    pub name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub plane: Option<SketchPlaneRecipe>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub plane_options: Option<SketchPlaneOptions>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub plane_feature: Option<crate::feature::FeatureId>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub orientation: Option<SketchOrientationDefinition>,
    pub geometry: SketchRecipe,
}
/// Collection schema and records; the enclosing document owns reservation marks.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DocumentSketchRecipe {
    pub version: u32,
    pub sketches: Vec<PlacedSketchRecipe>,
}
impl PlacedSketch {
    fn new(
        id: DocumentSketchId,
        name: &str,
        placement: SketchPlacement,
        geometry: Sketch2D,
    ) -> Self {
        Self {
            id,
            name: name.into(),
            placement,
            orientation: None,
            geometry,
            evaluation: None,
        }
    }
    /// Inspect the saved local sketch ID; example: `sketch.id()`.
    pub fn id(&self) -> DocumentSketchId {
        self.id
    }
    /// Inspect the authored name; example: `sketch.name()`.
    pub fn name(&self) -> &str {
        &self.name
    }
    /// Inspect the immutable geometry owner; example: `sketch.geometry().points()`.
    pub fn geometry(&self) -> &Sketch2D {
        &self.geometry
    }
    fn recipe(&self) -> PlacedSketchRecipe {
        let (plane, plane_options, plane_feature) = match self.placement {
            SketchPlacement::Fixed(frame) => (Some(frame.recipe()), Some(frame.options()), None),
            SketchPlacement::WorkPlane(id) => (None, None, Some(id)),
        };
        PlacedSketchRecipe {
            id: self.id,
            name: self.name.clone(),
            plane,
            plane_options,
            plane_feature,
            orientation: self.orientation,
            geometry: self.geometry.recipe(),
        }
    }
}
impl DocumentSketches {
    /// Compare exact authored contents for dirty/history decisions; example: `baseline.same_recipe(&candidate)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.sketches.len() == other.sketches.len()
            && self
                .sketches
                .iter()
                .zip(&other.sketches)
                .all(|(first, second)| {
                    first.id == second.id
                        && first.name == second.name
                        && first.placement.same_recipe(second.placement)
                        && orientation::same_orientation(first.orientation, second.orientation)
                        && first.geometry.same_recipe(&second.geometry)
                })
    }

    /// Compare recipes and retained verification for source consequences; example: `old.same_evaluated_state(&new)`.
    pub fn same_evaluated_state(&self, other: &Self) -> bool {
        self.same_recipe(other)
            && self
                .sketches
                .iter()
                .zip(&other.sketches)
                .all(|(first, second)| first.evaluation == second.evaluation)
    }

    /// Inspect records in creation order; example: `document.sketches()`.
    pub fn sketches(&self) -> &[PlacedSketch] {
        &self.sketches
    }
    /// Resolve a saved sketch ID; example: `document.get(DocumentSketchId(7))?`.
    pub fn get(&self, id: DocumentSketchId) -> Result<&PlacedSketch, SketchError> {
        self.sketches
            .iter()
            .find(|sketch| sketch.id == id)
            .ok_or_else(|| missing_sketch(id))
    }
    /// Add caller-reserved identities atomically; example: `document.add(id, "Sketch1", plane, geometry)?`.
    pub fn add(
        &mut self,
        id: DocumentSketchId,
        name: &str,
        plane: SketchPlane,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        self.insert_placed(id, name, SketchPlacement::Fixed(plane), geometry)
    }
    fn insert_placed(
        &mut self,
        id: DocumentSketchId,
        name: &str,
        placement: SketchPlacement,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        if id.0 == 0 || self.sketches.iter().any(|sketch| sketch.id == id) {
            return Err(SketchError::Definition(format!(
                "document sketch ID {id:?}: expected a unique nonzero sketch identity"
            )));
        }
        self.require_unique_entities(None, &geometry)?;
        self.sketches
            .push(PlacedSketch::new(id, name, placement, geometry));
        Ok(())
    }
    /// Remove a sketch without changing other identities; example: `document.remove(id)?`.
    pub fn remove(&mut self, id: DocumentSketchId) -> Result<PlacedSketch, SketchError> {
        let index = self.index(id)?;
        Ok(self.sketches.remove(index))
    }
    /// Rename without changing identity or geometry; example: `document.rename(id, "Mounting holes")?`.
    pub fn rename(&mut self, id: DocumentSketchId, name: &str) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].name = name.into();
        Ok(())
    }
    /// Publish edited or solved geometry after document identity checks; example: `document.replace_geometry(id, candidate)?`.
    pub fn replace_geometry(
        &mut self,
        id: DocumentSketchId,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.require_unique_entities(Some(id), &geometry)?;
        self.sketches[index].geometry = geometry;
        self.sketches[index].evaluation = None;
        Ok(())
    }
    /// Replace static placement without reauthoring 2D values; example: `document.replace_plane(id, SketchPlane::xz())?`.
    pub fn replace_plane(
        &mut self,
        id: DocumentSketchId,
        plane: SketchPlane,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].placement = SketchPlacement::Fixed(plane);
        Ok(())
    }
    /// Capture complete placed records in stable order; example: `document.recipe()`.
    pub fn recipe(&self) -> DocumentSketchRecipe {
        DocumentSketchRecipe {
            version: self.placement_recipe_version(),
            sketches: self.sketches.iter().map(PlacedSketch::recipe).collect(),
        }
    }
    fn placement_recipe_version(&self) -> u32 {
        if self
            .sketches
            .iter()
            .any(|placed| placed.orientation.is_some())
        {
            return 3;
        }
        if self
            .sketches
            .iter()
            .any(|placed| matches!(placed.placement, SketchPlacement::WorkPlane(_)))
        {
            return 2;
        }
        1
    }
    /// Restore a complete collection before replacing a document owner; example: `DocumentSketches::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &DocumentSketchRecipe) -> Result<Self, SketchError> {
        if !matches!(recipe.version, 1..=3) {
            return Err(SketchError::Definition(format!(
                "document sketch recipe version {}: expected 1, 2, or 3",
                recipe.version
            )));
        }
        let mut restored = Self::default();
        for sketch in &recipe.sketches {
            let placement = SketchPlacement::from_recipe(sketch, recipe.version)?;
            let geometry = Sketch2D::from_recipe(&sketch.geometry)?;
            restored.insert_placed(sketch.id, &sketch.name, placement, geometry)?;
            orientation::restore_orientation(&mut restored, sketch, recipe.version)?;
        }
        Ok(restored)
    }
    /// Inspect live sketch IDs for a separate reservation owner; example: `ids.restore(mark, &document.sketch_ids())`.
    pub fn sketch_ids(&self) -> Vec<u64> {
        self.sketches.iter().map(|sketch| sketch.id.0).collect()
    }
    /// Inspect globally unique entity IDs; constraint IDs stay local to their sketch.
    /// Example: `entity_ids.restore(mark, &document.entity_ids())`.
    pub fn entity_ids(&self) -> Vec<u64> {
        self.sketches
            .iter()
            .flat_map(|sketch| geometry_ids(&sketch.geometry))
            .collect()
    }
    fn index(&self, id: DocumentSketchId) -> Result<usize, SketchError> {
        self.sketches
            .iter()
            .position(|sketch| sketch.id == id)
            .ok_or_else(|| missing_sketch(id))
    }
    fn require_unique_entities(
        &self,
        replacing: Option<DocumentSketchId>,
        geometry: &Sketch2D,
    ) -> Result<(), SketchError> {
        let retained: BTreeSet<u64> = self
            .sketches
            .iter()
            .filter(|sketch| Some(sketch.id) != replacing)
            .flat_map(|sketch| geometry_ids(&sketch.geometry))
            .collect();
        for id in geometry_ids(geometry) {
            if retained.contains(&id) {
                return Err(SketchError::Definition(format!(
                    "document sketch entity ID {id}: expected one owner across all document sketches"
                )));
            }
        }
        Ok(())
    }
}
fn geometry_ids(geometry: &Sketch2D) -> Vec<u64> {
    geometry
        .points()
        .iter()
        .map(|point| point.id.0)
        .chain(geometry.lines().iter().map(|line| line.id.0))
        .chain(geometry.circles().iter().map(|circle| circle.id.0))
        .chain(geometry.arcs().iter().map(|arc| arc.id.0))
        .collect()
}
fn missing_sketch(id: DocumentSketchId) -> SketchError {
    SketchError::Definition(format!(
        "document sketch ID {id:?}: expected an existing sketch"
    ))
}

impl PartialEq for PlacedSketch {
    fn eq(&self, other: &Self) -> bool {
        // Derived verification is not authored content and is rebuilt after loading (#3722).
        self.id == other.id
            && self.name == other.name
            && self.placement == other.placement
            && self.orientation == other.orientation
            && self.geometry == other.geometry
    }
}
