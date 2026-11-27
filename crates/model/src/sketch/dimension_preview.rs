// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchDimensionalConstraintCreation, SketchError};
use crate::quantity::Quantity;

impl Sketch2D {
    /// Measure constructor intent with the same branch selection and measurement as an owned dimension.
    /// Example: `geometry.measure_dimension_input(&operands)?` before opening a value editor.
    pub fn measure_dimension_input(
        &self,
        input: &SketchDimensionalConstraintCreation,
    ) -> Result<Quantity, SketchError> {
        input.validate(self)?;
        let id = self.dimension_probe_identity()?;
        let mut candidate = self.clone();
        // Like ownership validation (#3787), the parameter is never resolved. Only this copy gets a relation.
        input.append(&mut candidate, id, 0)?;
        candidate.measure_dimension(id)
    }
    fn dimension_probe_identity(&self) -> Result<SketchConstraintId, SketchError> {
        let occupied: std::collections::BTreeSet<u64> = self
            .constraints()
            .iter()
            .map(|constraint| constraint.id.0)
            .collect();
        std::iter::successors(Some(1u64), |id| id.checked_add(1))
            .find(|id| !occupied.contains(id))
            .map(SketchConstraintId)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "dimension probe identity {}: expected a free positive u64 identity",
                    u64::MAX
                ))
            })
    }
}
