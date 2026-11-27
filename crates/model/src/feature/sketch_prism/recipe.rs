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
    pub taper: Option<DatumAngle>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub extent: Option<SketchPrismExtent>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cut_target: Option<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub join_target: Option<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub intersection_target: Option<Vec<u8>>,
}
impl SketchPrismDefinition {
    /// Capture only authored inputs; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> SketchPrismRecipe {
        let target_key = |operation| {
            self.boolean_target()
                .filter(|target| target.operation() == operation)
                .map(|target| target.key().to_vec())
        };
        let intersection_target = target_key(PrismBoolean::Intersect);
        SketchPrismRecipe {
            version: prism_recipe_version(
                self.extent.as_ref(),
                self.taper.is_some(),
                intersection_target.is_some(),
            ),
            sketch: self.sketch,
            selection: self.selection.clone(),
            displacement: self.displacement,
            reversed_normal: self.direction == SketchPrismDirection::Negative,
            centered_normal: self.direction == SketchPrismDirection::Symmetric,
            second_depth: self.second_depth,
            taper: self.taper,
            extent: self.extent.clone(),
            cut_target: target_key(PrismBoolean::Cut),
            join_target: target_key(PrismBoolean::Join),
            intersection_target,
        }
    }
    /// Validate the saved version and sources before adoption; example: `SketchPrismDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &SketchPrismRecipe) -> Result<Self, SketchPrismError> {
        let expected = prism_recipe_version(
            recipe.extent.as_ref(),
            recipe.taper.is_some(),
            recipe.intersection_target.is_some(),
        );
        if recipe.version != expected {
            return Err(SketchPrismError {
                sketch: recipe.sketch,
                cause: Box::new(SketchPrismCause::Definition(format!(
                    "prism recipe version {}: expected version {expected} for the saved prism inputs",
                    recipe.version
                ))),
            });
        }
        let definition = match &recipe.extent {
            Some(SketchPrismExtent::DistanceFromPlane { from }) => {
                Self::from_distance_recipe(recipe)?.with_start_plane(from.clone())
            }
            Some(extent) => Self::from_plane_recipe(recipe, extent),
            None => Self::from_distance_recipe(recipe),
        }?
        .with_taper(recipe.taper)?;
        definition.with_saved_target(recipe)
    }
    fn with_saved_target(self, recipe: &SketchPrismRecipe) -> Result<Self, SketchPrismError> {
        match (
            &recipe.cut_target,
            &recipe.join_target,
            &recipe.intersection_target,
        ) {
            (Some(key), None, None) => self.with_cut_target(key.clone()),
            (None, Some(key), None) => self.with_boolean_target(PrismBoolean::Join, key.clone()),
            (None, None, Some(key)) => {
                self.with_boolean_target(PrismBoolean::Intersect, key.clone())
            }
            (None, None, None)
                if matches!(self.extent(), Some(SketchPrismExtent::ThroughAll { .. })) =>
            {
                Err(self.decline(SketchPrismCause::Definition(
                    "through-all extent: expected a saved target body".into(),
                )))
            }
            (None, None, None) => Ok(self),
            _ => Err(self.decline(SketchPrismCause::Definition(
                "extrude recipe: expected only one Boolean target".into(),
            ))),
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

fn prism_recipe_version(
    extent: Option<&SketchPrismExtent>,
    tapered: bool,
    intersects: bool,
) -> u32 {
    if matches!(extent, Some(SketchPrismExtent::DistanceFromPlane { .. })) {
        return 4;
    }
    match (intersects, tapered) {
        (true, _) => 3,
        (false, true) => 2,
        (false, false) => 1,
    }
}
