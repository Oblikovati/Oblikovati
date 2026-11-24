// SPDX-License-Identifier: GPL-2.0-only
use super::*;
/// Versioned authored prism inputs; no derived state or body cache is serialized.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPrismRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub displacement: [DatumLength; 3],
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub reversed_normal: bool,
}
impl SketchPrismDefinition {
    /// Capture only authored inputs; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> SketchPrismRecipe {
        SketchPrismRecipe {
            version: 1,
            sketch: self.sketch,
            selection: self.selection.clone(),
            displacement: self.displacement,
            reversed_normal: self.reversed_normal,
        }
    }
    /// Validate the saved version and sources before adoption; example: `SketchPrismDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &SketchPrismRecipe) -> Result<Self, SketchPrismError> {
        if recipe.version != 1 {
            return Err(SketchPrismError {
                sketch: recipe.sketch,
                cause: Box::new(SketchPrismCause::Definition(format!(
                    "prism recipe version {}: expected version 1",
                    recipe.version
                ))),
            });
        }
        let definition = Self::new(recipe.sketch, recipe.selection.clone(), recipe.displacement)?;
        if recipe.reversed_normal {
            return definition.reverse_normal();
        }
        Ok(definition)
    }
}
