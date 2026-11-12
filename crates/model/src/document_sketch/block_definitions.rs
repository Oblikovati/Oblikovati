// SPDX-License-Identifier: GPL-2.0-only
//! The document's ordered registry of sketch block definitions.
use super::{
    DocumentSketchId, DocumentSketchRecipe, DocumentSketches, Sketch2D, SketchError, SketchRecipe,
};
use crate::sketch::SketchEntityId;

/// A named definition with local geometry; example: `definition.name()`.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchBlockDefinition {
    name: String,
    geometry: Sketch2D,
}
/// Saved definition; example: `SketchBlockDefinitionRecipe { name: "Rib".into(), geometry: None }`.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchBlockDefinitionRecipe {
    pub name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub geometry: Option<SketchRecipe>,
}
impl SketchBlockDefinition {
    /// Return the registry name; example: `assert_eq!(definition.name(), "Rib")`.
    pub fn name(&self) -> &str {
        &self.name
    }
    /// Return definition-local geometry; example: `definition.geometry().entity_ids()`.
    pub fn geometry(&self) -> &Sketch2D {
        &self.geometry
    }
}
impl DocumentSketches {
    /// Return definitions in creation order; example: `sketches.block_definitions()`.
    pub fn block_definitions(&self) -> &[SketchBlockDefinition] {
        &self.block_definitions
    }
    /// Add an empty definition; example: `sketches.define_empty_block("Rib")?`.
    pub fn define_empty_block(&mut self, name: &str) -> Result<usize, SketchError> {
        validate_name(name)?;
        if self.block_definitions.iter().any(|item| item.name == name) {
            return Err(SketchError::Definition(format!(
                "block definition {name:?}: expected a unique name"
            )));
        }
        let index = self.block_definitions.len();
        self.block_definitions.push(SketchBlockDefinition {
            name: name.into(),
            geometry: Sketch2D::default(),
        });
        Ok(index)
    }
    /// Capture a selection in definition-local identities without changing the source sketch.
    /// Example: `sketches.capture_block_definition("Rib", sketch, &[line])?;`.
    pub fn capture_block_definition(
        &mut self,
        name: &str,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
    ) -> Result<usize, SketchError> {
        let geometry = self.get(sketch)?.geometry().capture_selection(entities)?;
        self.define_block_geometry(name, geometry)
    }
    pub(crate) fn define_block_geometry(
        &mut self,
        name: &str,
        geometry: Sketch2D,
    ) -> Result<usize, SketchError> {
        let index = self.define_empty_block(name)?;
        self.block_definitions[index].geometry = geometry;
        Ok(index)
    }
    /// Delete an unplaced definition; example: `sketches.delete_block_definition("Rib")?`.
    pub fn delete_block_definition(&mut self, name: &str) -> Result<(), SketchError> {
        let index = self
            .block_definitions
            .iter()
            .position(|item| item.name == name)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "block definition {name:?}: expected an existing name"
                ))
            })?;
        let consumers: Vec<_> = self
            .sketches
            .iter()
            .filter(|sketch| {
                sketch
                    .block_instances()
                    .iter()
                    .any(|instance| instance.definition() == name)
            })
            .map(|sketch| sketch.name())
            .collect();
        if !consumers.is_empty() {
            return Err(SketchError::Definition(format!(
                "block definition {name:?} is used by sketches {consumers:?}: expected no placed instances before deletion"
            )));
        }
        self.block_definitions.remove(index);
        Ok(())
    }
    pub(super) fn block_definition_recipes(&self) -> Vec<SketchBlockDefinitionRecipe> {
        self.block_definitions
            .iter()
            .map(|item| SketchBlockDefinitionRecipe {
                name: item.name.clone(),
                geometry: (!item.geometry.owned_entity_ids().is_empty())
                    .then(|| item.geometry.recipe()),
            })
            .collect()
    }
    pub(super) fn restore_block_definitions(
        &mut self,
        recipe: &DocumentSketchRecipe,
    ) -> Result<(), SketchError> {
        if recipe.version < 8 && !recipe.block_definitions.is_empty() {
            return Err(SketchError::Definition(format!(
                "block definitions in recipe version {}: expected version 8 or later",
                recipe.version
            )));
        }
        for item in &recipe.block_definitions {
            if recipe.version < 9 && item.geometry.is_some() {
                return Err(SketchError::Definition(format!(
                    "block definition {:?} geometry in recipe version {}: expected version 9 or later",
                    item.name, recipe.version
                )));
            }
            let geometry = item
                .geometry
                .as_ref()
                .map(Sketch2D::from_recipe)
                .transpose()?
                .unwrap_or_default();
            if !geometry.constraints().is_empty() {
                return Err(SketchError::Definition(format!(
                    "block definition {:?} constraints: expected geometry without sketch constraints",
                    item.name
                )));
            }
            let index = self.define_empty_block(&item.name)?;
            self.block_definitions[index].geometry = geometry;
        }
        Ok(())
    }
    pub(super) fn same_block_recipes(&self, other: &Self) -> bool {
        self.block_definitions.len() == other.block_definitions.len()
            && self
                .block_definitions
                .iter()
                .zip(&other.block_definitions)
                .all(|(a, b)| a.name == b.name && a.geometry.same_recipe(&b.geometry))
    }
}
fn validate_name(name: &str) -> Result<(), SketchError> {
    if name.trim().is_empty() || name.contains('\0') {
        return Err(SketchError::Definition(format!(
            "block definition name {name:?}: expected nonempty text without NUL"
        )));
    }
    Ok(())
}
