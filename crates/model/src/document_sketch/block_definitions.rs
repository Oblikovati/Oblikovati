// SPDX-License-Identifier: GPL-2.0-only
//! The document's ordered registry of empty sketch block definitions.
use super::{DocumentSketchRecipe, DocumentSketches, SketchError};

/// A named definition awaiting geometry; example: `definition.name()`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchBlockDefinition {
    name: String,
}
/// Saved empty definition; example: `SketchBlockDefinitionRecipe { name: "Rib".into() }`.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchBlockDefinitionRecipe {
    pub name: String,
}
impl SketchBlockDefinition {
    /// Return the registry name; example: `assert_eq!(definition.name(), "Rib")`.
    pub fn name(&self) -> &str {
        &self.name
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
        self.block_definitions
            .push(SketchBlockDefinition { name: name.into() });
        Ok(index)
    }
    /// Delete an empty definition; example: `sketches.delete_empty_block("Rib")?`.
    pub fn delete_empty_block(&mut self, name: &str) -> Result<(), SketchError> {
        let index = self
            .block_definitions
            .iter()
            .position(|item| item.name == name)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "block definition {name:?}: expected an existing name"
                ))
            })?;
        self.block_definitions.remove(index);
        Ok(())
    }
    pub(super) fn block_definition_recipes(&self) -> Vec<SketchBlockDefinitionRecipe> {
        self.block_definitions
            .iter()
            .map(|item| SketchBlockDefinitionRecipe {
                name: item.name.clone(),
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
            self.define_empty_block(&item.name)?;
        }
        Ok(())
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
