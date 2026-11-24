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
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub centered_normal: bool,
}
impl SketchPrismDefinition {
    /// Capture only authored inputs; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> SketchPrismRecipe {
        SketchPrismRecipe {
            version: 1,
            sketch: self.sketch,
            selection: self.selection.clone(),
            displacement: self.displacement,
            reversed_normal: self.direction == SketchPrismDirection::Negative,
            centered_normal: self.direction == SketchPrismDirection::Symmetric,
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
        let direction = match (recipe.reversed_normal, recipe.centered_normal) {
            (false, false) => SketchPrismDirection::Positive,
            (true, false) => SketchPrismDirection::Negative,
            (false, true) => SketchPrismDirection::Symmetric,
            (true, true) => {
                return Err(SketchPrismError {
                    sketch: recipe.sketch,
                    cause: Box::new(SketchPrismCause::Definition(
                        "prism recipe: expected only one normal direction".into(),
                    )),
                });
            }
        };
        Self::new(recipe.sketch, recipe.selection.clone(), recipe.displacement)?
            .with_direction(direction)
    }
}
