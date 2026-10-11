// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchError, SketchPoint2D, compile};
use crate::parameter::ParameterCollection;
use oblikovati_solve::diagnostics::ConstraintStructure;
/// Available equation structure with every excluded target named; this is not solve success.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchAvailableTargetAnalysis {
    pub structure: ConstraintStructure,
    pub unavailable: Vec<SketchError>,
}
impl Sketch2D {
    /// Inspect DOF with unavailable driving targets quarantined; the authored sketch stays unchanged.
    /// Example: `sketch.analyze_available_targets(&parameters, &anchors, 1e-9)?` for a sick creation reply.
    pub fn analyze_available_targets(
        &self,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        rank_tolerance: f64,
    ) -> Result<SketchAvailableTargetAnalysis, SketchError> {
        let unavailable = self.unavailable_dimension_targets(parameters);
        let mut equations = self.clone();
        for failure in &unavailable {
            if let SketchError::Parameter { constraint, .. } = failure {
                equations.remove_dimensional_constraint(*constraint)?;
            }
        }
        let structure =
            equations.analyze_with_grounded_points(parameters, anchors, rank_tolerance)?;
        Ok(SketchAvailableTargetAnalysis {
            structure,
            unavailable,
        })
    }
    fn unavailable_dimension_targets(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Vec<SketchError> {
        self.dimensional_constraints()
            .filter_map(|(row, parameter)| {
                if self.reference_dimensions.contains(&row.id) {
                    return None;
                }
                compile::parameter_quantity(row, parameter, parameters).err()
            })
            .collect()
    }
}
