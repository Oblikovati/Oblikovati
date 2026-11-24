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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub second_depth: Option<DatumLength>,
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
            second_depth: self.second_depth,
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
        match (
            recipe.reversed_normal,
            recipe.centered_normal,
            recipe.second_depth,
        ) {
            (false, false, None) => Ok(definition),
            (true, false, None) => definition.with_direction(SketchPrismDirection::Negative),
            (false, true, None) => definition.with_direction(SketchPrismDirection::Symmetric),
            (false, false, Some(depth)) => definition.with_second_depth(depth),
            _ => Err(SketchPrismError {
                sketch: recipe.sketch,
                cause: Box::new(SketchPrismCause::Definition(
                    "prism recipe: expected only one normal direction".into(),
                )),
            }),
        }
    }
}
