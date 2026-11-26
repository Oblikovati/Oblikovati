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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub extent: Option<SketchPrismExtent>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cut_target: Option<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub join_target: Option<Vec<u8>>,
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
            extent: self.extent.clone(),
            cut_target: self.cut_target().map(|target| target.key().to_vec()),
            join_target: self
                .boolean_target()
                .filter(|target| target.operation() == PrismBoolean::Join)
                .map(|target| target.key().to_vec()),
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
        let definition = match &recipe.extent {
            Some(extent) => Self::from_plane_recipe(recipe, extent),
            None => Self::from_distance_recipe(recipe),
        }?;
        match (&recipe.cut_target, &recipe.join_target) {
            (Some(key), None) => definition.with_cut_target(key.clone()),
            (None, Some(key)) => definition.with_boolean_target(PrismBoolean::Join, key.clone()),
            (Some(_), Some(_)) => Err(definition.decline(SketchPrismCause::Definition(
                "extrude recipe: expected only one Boolean target".into(),
            ))),
            (None, None)
                if matches!(
                    definition.extent(),
                    Some(SketchPrismExtent::ThroughAll { .. })
                ) =>
            {
                Err(definition.decline(SketchPrismCause::Definition(
                    "through-all extent: expected a saved target body".into(),
                )))
            }
            (None, None) => Ok(definition),
        }
    }
    fn from_distance_recipe(recipe: &SketchPrismRecipe) -> Result<Self, SketchPrismError> {
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
    fn from_plane_recipe(
        recipe: &SketchPrismRecipe,
        extent: &SketchPrismExtent,
    ) -> Result<Self, SketchPrismError> {
        let mut definition =
            Self::between_planes(recipe.sketch, recipe.selection.clone(), extent.clone())?;
        definition.displacement = recipe.displacement;
        if recipe.reversed_normal || recipe.centered_normal || recipe.second_depth.is_some() {
            return Err(definition.decline(SketchPrismCause::Definition(
                "prism plane extent: expected plane bounds alone".into(),
            )));
        }
        validation::validate_plane_extent(&definition)
            .map_err(|cause| definition.decline(cause))?;
        Ok(definition)
    }
}
