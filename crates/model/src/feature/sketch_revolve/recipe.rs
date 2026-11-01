// SPDX-License-Identifier: GPL-2.0-only
use super::*;

/// Versioned authored revolve inputs with no solved geometry or native result.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRevolveRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub angle: DatumAngle,
}

impl SketchRevolveDefinition {
    /// Capture authored sources only; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> SketchRevolveRecipe {
        SketchRevolveRecipe {
            version: 1,
            sketch: self.sketch,
            selection: self.selection.clone(),
            angle: self.angle,
        }
    }

    /// Restore a validated authored recipe; example: `SketchRevolveDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &SketchRevolveRecipe) -> Result<Self, SketchRevolveError> {
        if recipe.version == 1 {
            return Self::new(recipe.sketch, recipe.selection.clone(), recipe.angle);
        }
        Err(SketchRevolveError {
            sketch: recipe.sketch,
            cause: Box::new(SketchRevolveCause::Definition(format!(
                "revolve recipe version {}: expected version 1",
                recipe.version
            ))),
        })
    }
}
