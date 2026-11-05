// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchDimensionLimits, SketchError},
};
/// Optional authored changes; omitted fields preserve the existing dimension state.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct SketchDimensionEdit {
    pub target_expression: Option<String>,
    pub reference: Option<bool>,
    pub drive_limits: Option<SketchDimensionLimits>,
}
impl DocumentSnapshot {
    /// Edit one dimension on an isolated candidate; expression drive stores a clamped constant as in Go.
    /// Example: `snapshot.edit_sketch_dimension(sketch, dimension, &edit)?;` inside an edit.
    pub fn edit_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), SketchError> {
        let mut candidate = self.clone();
        candidate.apply_dimension_edit(sketch, dimension, edit)?;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(())
    }
    fn apply_dimension_edit(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), SketchError> {
        self.sketches
            .get(sketch)?
            .geometry()
            .dimension_source(dimension)?;
        if let Some(limits) = edit.drive_limits {
            self.sketches
                .set_dimension_limits(sketch, dimension, limits)?;
        }
        if let Some(reference) = edit.reference {
            self.sketches
                .set_dimension_reference(sketch, dimension, reference)?;
        }
        let Some(source) = &edit.target_expression else {
            return Ok(());
        };
        self.drive_dimension_target(sketch, dimension, source)
    }
    fn drive_dimension_target(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        source: &str,
    ) -> Result<(), SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        let (parameter, category) = geometry.dimension_source(dimension)?;
        self.parameters
            .validate_owned_scalar(parameter)
            .map_err(|cause| drive_error(parameter, source, cause))?;
        let mut target = self
            .parameters
            .evaluate_scalar_expression(source, category)
            .map_err(|cause| drive_error(parameter, source, cause))?;
        if let Some(limits) = geometry.dimension_limits(dimension)? {
            target.value = limits.clamp(target.value);
        }
        self.parameters
            .set_value(parameter, target)
            .map_err(|cause| drive_error(parameter, source, cause))
    }
}
fn drive_error(parameter: u64, source: &str, cause: impl std::fmt::Display) -> SketchError {
    SketchError::Definition(format!(
        "dimension parameter {parameter} drive expression {source:?}: expected a finite compatible owned Model scalar target: {cause}"
    ))
}
