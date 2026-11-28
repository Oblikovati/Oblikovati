// SPDX-License-Identifier: GPL-2.0-only
use super::*;

/// Persisted operation and body reference key for a Revolve Boolean.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RevolveBooleanRecipe {
    pub operation: PrismBoolean,
    pub key: Vec<u8>,
}

/// Versioned authored revolve inputs with no solved geometry or native result.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRevolveRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub axis: Option<SketchEntityId>,
    pub angle: DatumAngle,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub second_angle: Option<DatumAngle>,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub full: bool,
    #[serde(default, skip_serializing_if = "SketchRevolveDirection::is_positive")]
    pub direction: SketchRevolveDirection,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub surface: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub boolean_target: Option<RevolveBooleanRecipe>,
}

impl SketchRevolveDefinition {
    /// Capture authored sources only; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> SketchRevolveRecipe {
        let is_v1 = self.axis.is_none()
            && self.second_angle.is_none()
            && !self.full
            && self.direction == SketchRevolveDirection::Positive
            && !self.surface
            && self.boolean_target.is_none();
        SketchRevolveRecipe {
            version: if is_v1 { 1 } else { 2 },
            sketch: self.sketch,
            selection: self.selection.clone(),
            axis: self.axis,
            angle: self.angle,
            second_angle: self.second_angle,
            full: self.full,
            direction: self.direction,
            surface: self.surface,
            boolean_target: self
                .boolean_target
                .as_ref()
                .map(|(recipe, _)| recipe.clone()),
        }
    }

    /// Restore a validated authored recipe; example: `SketchRevolveDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &SketchRevolveRecipe) -> Result<Self, SketchRevolveError> {
        let is_v1_shape = recipe.axis.is_none()
            && recipe.second_angle.is_none()
            && !recipe.full
            && recipe.direction == SketchRevolveDirection::Positive
            && !recipe.surface
            && recipe.boolean_target.is_none();
        match recipe.version {
            1 if is_v1_shape => Self::new(recipe.sketch, recipe.selection.clone(), recipe.angle),
            2 => {
                if recipe.surface && recipe.boolean_target.is_some() {
                    return Err(SketchRevolveError {
                        sketch: recipe.sketch,
                        cause: Box::new(SketchRevolveCause::Definition(
                            "surface Revolve: expected independent output without a Boolean target"
                                .into(),
                        )),
                    });
                }
                let mut definition =
                    Self::new(recipe.sketch, recipe.selection.clone(), recipe.angle)?;
                if let Some(axis) = recipe.axis {
                    definition = definition.with_axis(axis)?;
                }
                if recipe.full {
                    definition = definition.with_full_turn();
                } else if let Some(second) = recipe.second_angle {
                    definition = definition.with_asymmetric_angle(second)?;
                } else {
                    definition = definition.with_direction(recipe.direction);
                }
                if recipe.surface {
                    definition = definition.with_surface_output(true);
                } else if let Some(target) = &recipe.boolean_target {
                    definition =
                        definition.with_boolean_target(target.operation, target.key.clone())?;
                }
                Ok(definition)
            }
            _ => Err(SketchRevolveError {
                sketch: recipe.sketch,
                cause: Box::new(SketchRevolveCause::Definition(format!(
                    "revolve recipe version {}: expected version 1 or 2",
                    recipe.version
                ))),
            }),
        }
    }
}
