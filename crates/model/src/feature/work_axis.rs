// SPDX-License-Identifier: GPL-2.0-only
//! Checked datum line values from Go work_axis_point.go, aa8297d1; ownership stays in the feature program.
mod error;
mod reference;
use crate::sketch::{
    SketchPlaneOptions, normalize_direction, point_difference, require_unit, same_components,
};
pub use error::WorkAxisError;
pub use reference::WorkAxisReference;

/// An immutable finite datum line; its direction is dimensionless and normalized.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct WorkAxisLine {
    origin: [f64; 3],
    direction: [f64; 3],
}
/// Versioned source value, never a feature ID or a cached evaluation result.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkAxisLineRecipe {
    pub version: u32,
    pub origin: [f64; 3],
    pub direction: [f64; 3],
}
impl WorkAxisLine {
    /// Check a finite origin and normalize one nonzero direction; example: `WorkAxisLine::new([1.0,2.0,3.0], [0.0,0.0,7.0])?`.
    pub fn new(origin: [f64; 3], direction: [f64; 3]) -> Result<Self, WorkAxisError> {
        require_axis_origin(origin)?;
        let unit = normalize_direction(direction)
            .map_err(|cause| WorkAxisError::Direction { direction, cause })?;
        Ok(Self {
            origin,
            direction: unit,
        })
    }
    /// Resolve a directed line through two current points; example: `WorkAxisLine::through_points([0.0;3], [0.0,0.0,4.0])?`.
    pub fn through_points(first: [f64; 3], second: [f64; 3]) -> Result<Self, WorkAxisError> {
        require_axis_origin(first)?;
        require_axis_origin(second)?;
        Self::new(first, point_difference(first, second))
    }
    /// Inspect the origin in document working units; example: `axis.origin()`.
    pub fn origin(self) -> [f64; 3] {
        self.origin
    }
    /// Inspect the normalized current line direction; example: `axis.direction()` for sketch orientation.
    pub fn direction(self) -> [f64; 3] {
        self.direction
    }
    /// Capture the checked source value for an authored line definition; example: `axis.recipe()`.
    pub fn recipe(self) -> WorkAxisLineRecipe {
        WorkAxisLineRecipe {
            version: 1,
            origin: self.origin,
            direction: self.direction,
        }
    }
    /// Validate saved unit components without renormalizing or changing bits; example: `WorkAxisLine::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkAxisLineRecipe) -> Result<Self, WorkAxisError> {
        if recipe.version != 1 {
            return Err(WorkAxisError::RecipeVersion(recipe.version));
        }
        require_axis_origin(recipe.origin)?;
        require_unit(
            recipe.direction,
            SketchPlaneOptions::default().unit_deviation,
        )
        .map_err(|cause| WorkAxisError::Direction {
            direction: recipe.direction,
            cause,
        })?;
        Ok(Self {
            origin: recipe.origin,
            direction: recipe.direction,
        })
    }
    /// Compare exact source bits for history, including signed origin coordinates; example: `before.same_recipe(after)`.
    pub fn same_recipe(self, other: Self) -> bool {
        same_components(self.origin, other.origin)
            && same_components(self.direction, other.direction)
    }
}
fn require_axis_origin(origin: [f64; 3]) -> Result<(), WorkAxisError> {
    if !origin.iter().all(|value| value.is_finite()) {
        return Err(WorkAxisError::Origin(origin));
    }
    Ok(())
}
