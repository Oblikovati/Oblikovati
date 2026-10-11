// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, sketch_relation_create::solved_sketch_dof};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchError},
};
use oblikovati_solve::nonlinear::SolveOptions;
impl DocumentSnapshot {
    /// Remove an owned dimension and its unshared source atomically, then measure resulting DOF.
    /// Example: `snapshot.delete_sketch_dimension(sketch, constraint, Default::default())?;`.
    pub fn delete_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
        options: SolveOptions,
    ) -> Result<usize, SketchError> {
        let parameter = self.owned_dimension_parameter(sketch, constraint)?;
        let mut candidate = self.clone();
        let mut geometry = candidate.sketches.get(sketch)?.geometry().clone();
        geometry.remove_dimensional_constraint(constraint)?;
        candidate.sketches.replace_geometry(sketch, geometry)?;
        candidate.remove_retired_dimension_sources([parameter].into())?;
        let solved = candidate.solve_one_document_sketch(sketch, options, || false)?;
        let dof = solved_sketch_dof(&solved, sketch)?;
        self.parameters = candidate.parameters;
        self.sketches = solved.candidate;
        Ok(dof)
    }
    fn owned_dimension_parameter(
        &self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<u64, SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        let parameter = geometry.dimensional_constraints()
            .find(|(row, _)| row.id == constraint).map(|(_, parameter)| parameter)
            .ok_or_else(|| SketchError::Definition(format!(
                "sketch constraint {constraint:?}: expected an owned dimensional relation for deletion"
            )))?;
        self.parameters
            .validate_owned_scalar(parameter)
            .map_err(|cause| source_error(parameter, cause))?;
        Ok(parameter)
    }
}
fn source_error(parameter: u64, cause: crate::parameter::ParameterError<u64>) -> SketchError {
    SketchError::Definition(format!(
        "dimension parameter {parameter}: expected an owned Model scalar source for deletion: {cause}"
    ))
}
