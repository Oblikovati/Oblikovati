// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{
    PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
    tangent::select_signed_side,
};
impl Sketch2D {
    /// Retain the current offset side before binding its magnitude source.
    /// Example: `sketch.add_offset_spline_dimension(id, offset_id, parameter_id)?`.
    pub fn add_offset_spline_dimension(
        &mut self,
        id: SketchConstraintId,
        offset: SketchEntityId,
        parameter: u64,
    ) -> Result<(), SketchError> {
        let distance = self.offset_spline(offset)?.distance();
        let side = select_signed_side(distance).map_err(|cause| SketchError::Definition(format!("offset spline {offset:?} seed distance {distance}: expected a selected nonzero side or an explicit side relation: {cause}")))?;
        self.add_constraint(
            id,
            PlanarConstraint::OffsetSplineDistance {
                offset,
                parameter,
                side,
            },
        )
    }
}
pub(super) fn has_offset_dimension(recipe: &crate::sketch::SketchRecipe) -> bool {
    recipe.constraints.iter().any(|constraint| {
        matches!(
            constraint.relation,
            PlanarConstraint::OffsetSplineDistance { .. }
        )
    })
}
pub(super) fn require_offset_dimension_schema(
    recipe: &crate::sketch::SketchRecipe,
) -> Result<(), SketchError> {
    if recipe.version < 11 && has_offset_dimension(recipe) {
        return Err(SketchError::Definition(format!(
            "offset-spline dimension recipe version {}: expected version 11 for retained side metadata",
            recipe.version
        )));
    }
    Ok(())
}
