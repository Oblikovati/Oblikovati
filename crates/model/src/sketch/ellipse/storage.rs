// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{Sketch2D, SketchRecipe};
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct EllipseDefinition {
    id: SketchEntityId,
    center: SketchEntityId,
    major_axis: [f64; 2],
    radii: [f64; 2],
}
impl TryFrom<EllipseDefinition> for SketchEllipse2D {
    type Error = SketchError;
    fn try_from(definition: EllipseDefinition) -> Result<Self, SketchError> {
        require_ellipse_identities(definition.id, definition.center)?;
        require_ellipse_radii(definition.radii)?;
        require_saved_axis(definition.id, definition.major_axis)?;
        Ok(Self {
            id: definition.id,
            center: definition.center,
            major_axis: definition.major_axis,
            radii: definition.radii,
        })
    }
}
fn require_saved_axis(id: SketchEntityId, axis: [f64; 2]) -> Result<(), SketchError> {
    let length = axis[0].hypot(axis[1]);
    // Stored unit directions keep their exact bits; this bound measures normalization roundoff only.
    if !length.is_finite() || (length - 1.0).abs() > 8.0 * f64::EPSILON {
        return Err(SketchError::Definition(format!(
            "ellipse {id:?} saved axis {axis:?}: expected a finite unit direction"
        )));
    }
    Ok(())
}
impl Sketch2D {
    pub(crate) fn capture_ellipse_schema(&self, recipe: &mut SketchRecipe) {
        if self.ellipses.is_empty() {
            return;
        }
        recipe.version = 7;
        recipe.ellipses = Some(self.ellipses.clone());
        self.capture_dimension_schema_metadata(recipe);
    }
    pub(crate) fn restore_ellipses(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        require_minor_dimension_schema(recipe)?;
        match (recipe.version, &recipe.ellipses) {
            (1..=6, None) => Ok(()),
            (7..=11, Some(shapes)) => shapes.iter().try_for_each(|shape| self.add_ellipse(*shape)),
            shape => Err(SketchError::Definition(format!(
                "ellipse recipe {shape:?}: expected no ellipses in versions 1-6 or explicit ellipse values in versions 7-11"
            ))),
        }
    }
    pub(crate) fn capture_ellipse_dimension_schema(&self, recipe: &mut SketchRecipe) {
        if !has_minor_dimension(recipe) {
            return;
        }
        recipe.version = 9;
        self.capture_complete_schema_metadata(recipe);
    }
}

fn has_minor_dimension(recipe: &SketchRecipe) -> bool {
    recipe.constraints.iter().any(|constraint| {
        matches!(
            constraint.relation,
            crate::sketch::PlanarConstraint::EllipseMinorRadius { .. }
        )
    })
}
fn require_minor_dimension_schema(recipe: &SketchRecipe) -> Result<(), SketchError> {
    if recipe.version < 9 && has_minor_dimension(recipe) {
        return Err(SketchError::Definition(format!(
            "ellipse minor-radius recipe version {}: expected version 9, version 10, or version 11",
            recipe.version
        )));
    }
    Ok(())
}
