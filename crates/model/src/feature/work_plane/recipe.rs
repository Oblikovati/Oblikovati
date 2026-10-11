// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

/// Versioned authored construction data; no evaluated caches are stored.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneRecipe {
    pub version: u32,
    pub definition: WorkPlaneDefinitionRecipe,
}

/// One serialized definition selected by variant; scalar closures are not saved.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPlaneDefinitionRecipe {
    Axis(crate::feature::work_axis::WorkAxisLineRecipe),
    Fixed {
        frame: SketchPlaneRecipe,
        options: SketchPlaneOptions,
    },
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
}

impl WorkPlaneDefinition {
    /// Capture authored source without evaluated output; example: `definition.recipe()`.
    pub fn recipe(self) -> WorkPlaneRecipe {
        let definition = match self {
            Self::Axis(axis) => WorkPlaneDefinitionRecipe::Axis(axis.recipe()),
            Self::Fixed(plane) => WorkPlaneDefinitionRecipe::Fixed {
                frame: plane.recipe(),
                options: plane.options(),
            },
            Self::Offset { base, distance } => WorkPlaneDefinitionRecipe::Offset { base, distance },
        };
        WorkPlaneRecipe {
            version: if matches!(self, Self::Axis(_)) { 2 } else { 1 },
            definition,
        }
    }

    /// Restore and validate a supported recipe; example: `WorkPlaneDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPlaneRecipe) -> Result<Self, WorkPlaneError> {
        let expected = if matches!(recipe.definition, WorkPlaneDefinitionRecipe::Axis(_)) {
            2
        } else {
            1
        };
        if recipe.version != expected {
            return Err(WorkPlaneError::RecipeVersion(recipe.version));
        }
        restore_definition(recipe.definition)
    }
}

fn restore_definition(
    definition: WorkPlaneDefinitionRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    match definition {
        WorkPlaneDefinitionRecipe::Axis(axis) => {
            crate::feature::work_axis::WorkAxisLine::from_recipe(axis)
                .map(WorkPlaneDefinition::Axis)
                .map_err(WorkPlaneError::Axis)
        }
        WorkPlaneDefinitionRecipe::Fixed { frame, options } => {
            SketchPlane::from_recipe(frame, options)
                .map(WorkPlaneDefinition::Fixed)
                .map_err(WorkPlaneError::Frame)
        }
        WorkPlaneDefinitionRecipe::Offset { base, distance } => validate_offset(base, distance),
    }
}

fn validate_offset(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    if let WorkPlaneDistance::Constant(value) = distance
        && !value.is_finite()
    {
        return Err(WorkPlaneError::Offset {
            parameter: None,
            quantity: crate::quantity::Quantity::new(value, crate::quantity::Dimension::Length),
        });
    }
    Ok(WorkPlaneDefinition::Offset { base, distance })
}
