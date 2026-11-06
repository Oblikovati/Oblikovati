// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::{
    feature::{FeatureId, FeatureReadKey},
    sketch::{SketchError, SketchFrameAxis, SketchPlane},
};

/// A saved reference retains identity rather than its first resolved direction.
/// Keep existing sketch call sites and saved variant names while sharing the datum reference owner.
pub use crate::feature::work_axis::WorkAxisReference as SketchAxisReference;
/// Authored model inputs; the public wire DTO remains owned by Oblikovati.API.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchOrientationDefinition {
    pub reference: SketchAxisReference,
    pub sketch_axis: SketchFrameAxis,
    pub reverse: bool,
    pub origin: Option<[f64; 3]>,
}
/// Inject the current feature-axis catalogue; it must not return a cached last-good direction.
pub trait SketchAxisResolver {
    /// Resolve one current stable identity; example: `axes.axis_direction(FeatureId(7))?`.
    fn axis_direction(&self, feature: FeatureId) -> Result<[f64; 3], SketchError>;
}
pub(crate) struct UnavailableSketchAxes;
impl SketchAxisResolver for UnavailableSketchAxes {
    fn axis_direction(&self, feature: FeatureId) -> Result<[f64; 3], SketchError> {
        Err(SketchError::Definition(format!(
            "sketch orientation axis {feature:?}: expected an available current axis resolver"
        )))
    }
}
impl PlacedSketch {
    /// Inspect the saved orientation inputs; example: `placed.orientation()`.
    pub fn orientation(&self) -> Option<SketchOrientationDefinition> {
        self.orientation
    }
}
impl DocumentSketches {
    /// Set checked authored inputs without changing local geometry or solve stamps; example: `sketches.set_orientation(id, Some(definition))?`.
    pub fn set_orientation(
        &mut self,
        id: DocumentSketchId,
        orientation: Option<SketchOrientationDefinition>,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        if let Some(definition) = orientation {
            validate_orientation(definition)?;
        }
        self.sketches[index].orientation = orientation;
        Ok(())
    }
}
impl SketchOrientationDefinition {
    pub(super) fn read_keys(self) -> Vec<FeatureReadKey> {
        match self.reference {
            SketchAxisReference::Feature(id) => vec![FeatureReadKey::Feature(id)],
            _ => Vec::new(),
        }
    }
    fn direction(self, axes: &dyn SketchAxisResolver) -> Result<[f64; 3], SketchError> {
        match self.reference {
            SketchAxisReference::OriginX => Ok([1.0, 0.0, 0.0]),
            SketchAxisReference::OriginY => Ok([0.0, 1.0, 0.0]),
            SketchAxisReference::OriginZ => Ok([0.0, 0.0, 1.0]),
            SketchAxisReference::Feature(id) => axes.axis_direction(id),
        }
    }
}
pub(super) fn orient_current(
    base: SketchPlane,
    orientation: Option<SketchOrientationDefinition>,
    axes: &dyn SketchAxisResolver,
) -> Result<SketchPlane, SketchError> {
    let Some(definition) = orientation else {
        return Ok(base);
    };
    base.with_orientation(
        definition.direction(axes)?,
        definition.sketch_axis,
        definition.reverse,
        definition.origin,
    )
}
pub(super) fn same_orientation(
    first: Option<SketchOrientationDefinition>,
    second: Option<SketchOrientationDefinition>,
) -> bool {
    match (first, second) {
        (Some(first), Some(second)) => {
            first.reference == second.reference
                && first.sketch_axis == second.sketch_axis
                && first.reverse == second.reverse
                && first.origin.map(|origin| origin.map(f64::to_bits))
                    == second.origin.map(|origin| origin.map(f64::to_bits))
        }
        (None, None) => true,
        _ => false,
    }
}
pub(super) fn restore_orientation(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if recipe.orientation.is_some() && version < 3 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} orientation in recipe version {version}: expected version 3",
            recipe.id
        )));
    }
    sketches.set_orientation(recipe.id, recipe.orientation)
}
fn validate_orientation(definition: SketchOrientationDefinition) -> Result<(), SketchError> {
    let zero_reference = definition.reference == SketchAxisReference::Feature(FeatureId(0));
    let invalid_origin = definition
        .origin
        .is_some_and(|origin| !origin.iter().all(|value| value.is_finite()));
    if zero_reference || invalid_origin {
        return Err(SketchError::Definition(format!(
            "sketch orientation {definition:?}: expected a nonzero feature reference and three finite origin components"
        )));
    }
    Ok(())
}
