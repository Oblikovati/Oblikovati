// SPDX-License-Identifier: GPL-2.0-only
//! A sketch's placed references to document-owned block definitions.
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe, SketchError};
use crate::sketch::SketchEntityId;

/// A definition placed in sketch coordinates; example: `SketchBlockInstance::new(id, "Rib", [0., 0.], 0., 1.)?`.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchBlockInstance {
    id: SketchEntityId,
    definition: String,
    position: [f64; 2],
    rotation: f64,
    scale: f64,
}
impl SketchBlockInstance {
    /// Validate one placement; example: `SketchBlockInstance::new(id, "Rib", [1., 2.], 0., 1.)?`.
    pub fn new(
        id: SketchEntityId,
        definition: &str,
        position: [f64; 2],
        rotation: f64,
        scale: f64,
    ) -> Result<Self, SketchError> {
        let instance = Self {
            id,
            definition: definition.into(),
            position,
            rotation,
            scale,
        };
        instance.validate()?;
        Ok(instance)
    }
    fn validate(&self) -> Result<(), SketchError> {
        if self.id.0 == 0
            || self.definition.trim().is_empty()
            || self.definition.contains('\0')
            || self.position.iter().any(|value| !value.is_finite())
            || !self.rotation.is_finite()
            || !self.scale.is_finite()
            || self.scale == 0.
        {
            return Err(SketchError::Definition(format!(
                "block instance {self:?}: expected a nonzero identity, a definition name, finite position and rotation, and a finite nonzero scale"
            )));
        }
        Ok(())
    }
    /// Return the document-wide entity identity; example: `instance.id()`.
    pub fn id(&self) -> SketchEntityId {
        self.id
    }
    /// Return the definition name; example: `instance.definition()`.
    pub fn definition(&self) -> &str {
        &self.definition
    }
    /// Return the placement origin; example: `instance.position()`.
    pub fn position(&self) -> [f64; 2] {
        self.position
    }
    /// Return the counterclockwise rotation in radians; example: `instance.rotation()`.
    pub fn rotation(&self) -> f64 {
        self.rotation
    }
    /// Return the uniform scale; example: `instance.scale()`.
    pub fn scale(&self) -> f64 {
        self.scale
    }
    /// Map a definition point into its sketch; example: `instance.to_sketch([0., 0.])?`.
    pub fn to_sketch(&self, [x, y]: [f64; 2]) -> Result<[f64; 2], SketchError> {
        let (sin, cos) = self.rotation.sin_cos();
        let point = [
            self.position[0] + self.scale * (cos * x - sin * y),
            self.position[1] + self.scale * (sin * x + cos * y),
        ];
        self.check_mapped_point([x, y], point)
    }
    /// Map a sketch point into the definition; example: `instance.to_definition([1., 2.])?`.
    pub fn to_definition(&self, point: [f64; 2]) -> Result<[f64; 2], SketchError> {
        let (sin, cos) = self.rotation.sin_cos();
        let x = (point[0] - self.position[0]) / self.scale;
        let y = (point[1] - self.position[1]) / self.scale;
        self.check_mapped_point(point, [cos * x + sin * y, -sin * x + cos * y])
    }
    fn check_mapped_point(
        &self,
        source: [f64; 2],
        mapped: [f64; 2],
    ) -> Result<[f64; 2], SketchError> {
        if source.iter().chain(&mapped).any(|value| !value.is_finite()) {
            return Err(SketchError::Definition(format!(
                "block instance {:?} point {source:?}: expected finite coordinates in both spaces",
                self.id
            )));
        }
        Ok(mapped)
    }
}
impl PlacedSketch {
    /// Return placed instances in creation order; example: `sketch.block_instances()`.
    pub fn block_instances(&self) -> &[SketchBlockInstance] {
        &self.block_instances
    }
}
impl DocumentSketches {
    pub(crate) fn remove_block_instances(
        &mut self,
        sketch: DocumentSketchId,
        selected: &[SketchEntityId],
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let index = self.index(sketch)?;
        let instances = &mut self.sketches[index].block_instances;
        for id in selected {
            if !instances.iter().any(|instance| instance.id() == *id) {
                return Err(SketchError::Definition(format!(
                    "block instance {id:?} in sketch {sketch:?}: expected a current placed instance"
                )));
            }
        }
        let removed = instances
            .iter()
            .filter(|instance| selected.contains(&instance.id()))
            .map(|instance| instance.id())
            .collect();
        instances.retain(|instance| !selected.contains(&instance.id()));
        Ok(removed)
    }
    /// Count references across the document; example: `sketches.block_instance_count("Rib")`.
    pub fn block_instance_count(&self, definition: &str) -> usize {
        self.sketches
            .iter()
            .flat_map(|sketch| &sketch.block_instances)
            .filter(|instance| instance.definition() == definition)
            .count()
    }
    /// Insert a reserved instance after checking its owner and definition.
    /// Example: `sketches.insert_block_instance(sketch, instance)?;`.
    pub fn insert_block_instance(
        &mut self,
        sketch: DocumentSketchId,
        instance: SketchBlockInstance,
    ) -> Result<(), SketchError> {
        instance.validate()?;
        let index = self.index(sketch)?;
        if !self
            .block_definitions
            .iter()
            .any(|item| item.name() == instance.definition())
        {
            return Err(SketchError::Definition(format!(
                "block instance {:?} definition {:?}: expected an existing definition",
                instance.id(),
                instance.definition()
            )));
        }
        if self.entity_ids().contains(&instance.id().0) {
            return Err(SketchError::Definition(format!(
                "block instance ID {:?}: expected a unique document entity identity",
                instance.id()
            )));
        }
        self.sketches[index].block_instances.push(instance);
        Ok(())
    }
    pub(super) fn restore_block_instances(
        &mut self,
        recipe: &PlacedSketchRecipe,
        version: u32,
    ) -> Result<(), SketchError> {
        if version < 10 && !recipe.block_instances.is_empty() {
            return Err(SketchError::Definition(format!(
                "sketch {:?} block instances in recipe version {version}: expected version 10 or later",
                recipe.id
            )));
        }
        for instance in &recipe.block_instances {
            self.insert_block_instance(recipe.id, instance.clone())?;
        }
        Ok(())
    }
}
pub(super) fn same_instances(
    first: &[SketchBlockInstance],
    second: &[SketchBlockInstance],
) -> bool {
    first.len() == second.len()
        && first.iter().zip(second).all(|(a, b)| {
            a.id == b.id
                && a.definition == b.definition
                && a.position
                    .iter()
                    .zip(b.position)
                    .all(|(x, y)| x.to_bits() == y.to_bits())
                && a.rotation.to_bits() == b.rotation.to_bits()
                && a.scale.to_bits() == b.scale.to_bits()
        })
}
