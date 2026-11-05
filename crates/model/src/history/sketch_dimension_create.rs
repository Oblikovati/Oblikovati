// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, sketch_relation_create::solved_sketch_dof};
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketchSolveOutcome},
    quantity::Quantity,
    sketch::{Sketch2D, SketchConstraintId, SketchDimensionalConstraintCreation, SketchError},
};
use oblikovati_solve::nonlinear::SolveOptions;
mod error;
pub use error::SketchDimensionCreationError;
/// One authored dimension with a fresh owned Model formula and optional creation metadata.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchDimensionCreation {
    pub operands: SketchDimensionalConstraintCreation,
    pub expression: String,
    pub reference: bool,
    pub text_point: Option<[f64; 2]>,
}
/// Current measured reply; unavailable targets remain explicit and do not imply convergence.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchDimensionCreationReport {
    pub constraint: SketchConstraintId,
    pub parameter: u64,
    pub parameter_name: String,
    pub dimension_index: usize,
    pub measured: Quantity,
    pub degrees_of_freedom: usize,
    pub unavailable_targets: Vec<SketchError>,
}
struct ReservedDimension {
    constraint: SketchConstraintId,
    parameter: u64,
    parameter_name: String,
}
impl DocumentSnapshot {
    /// Create the source, relation and options together on an isolated candidate before solving.
    /// Example: `snapshot.create_sketch_dimension(sketch, &input, options, ids)?` in a document edit.
    pub fn create_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchDimensionCreationReport, SketchDimensionCreationError> {
        let geometry = self.prepare_dimension_creation(sketch, input, options)?;
        let reserved = self.reserve_dimension_creation(sketch, ids)?;
        let mut candidate = self.clone();
        candidate.append_owned_dimension(sketch, input, geometry, &reserved)?;
        let report = candidate.evaluate_created_dimension(sketch, &reserved, options)?;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(report)
    }
    fn prepare_dimension_creation(
        &self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        options: SolveOptions,
    ) -> Result<Sketch2D, SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        input.operands.validate(geometry)?;
        if let Some(position) = input.text_point {
            crate::sketch::require_position(position)?;
        }
        if input.operands.requires_current_seed() {
            return self.current_sketch_authoring_geometry(sketch, options);
        }
        Ok(geometry.clone())
    }
    fn reserve_dimension_creation(
        &self,
        sketch: DocumentSketchId,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<ReservedDimension, SketchDimensionCreationError> {
        let parameter_name = ids.allocate_dimension_name(sketch, &self.parameters)?;
        let parameter = ids.allocate()?;
        let constraint = ids.allocate_sketch_constraint(sketch)?;
        Ok(ReservedDimension {
            constraint,
            parameter,
            parameter_name,
        })
    }
    fn append_owned_dimension(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        mut geometry: Sketch2D,
        reserved: &ReservedDimension,
    ) -> Result<(), SketchDimensionCreationError> {
        self.parameters.insert_scalar_expression(
            reserved.parameter,
            &reserved.parameter_name,
            &input.expression,
            input.operands.dimension(),
        )?;
        input
            .operands
            .append(&mut geometry, reserved.constraint, reserved.parameter)?;
        geometry.set_dimension_reference(reserved.constraint, input.reference)?;
        if let Some(position) = input.text_point {
            geometry.set_dimension_text_point(reserved.constraint, position)?;
        }
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok(())
    }
    fn evaluate_created_dimension(
        &mut self,
        sketch: DocumentSketchId,
        reserved: &ReservedDimension,
        options: SolveOptions,
    ) -> Result<SketchDimensionCreationReport, SketchError> {
        let solved = self.solve_one_document_sketch(sketch, options, || false)?;
        let (degrees_of_freedom, unavailable_targets) =
            self.creation_dimension_structure(sketch, &solved, options)?;
        self.sketches = solved.candidate;
        let geometry = self.sketches.get(sketch)?.geometry();
        Ok(SketchDimensionCreationReport {
            constraint: reserved.constraint,
            parameter: reserved.parameter,
            parameter_name: reserved.parameter_name.clone(),
            dimension_index: geometry.dimensional_constraints().count() - 1,
            measured: geometry.measure_dimension(reserved.constraint)?,
            degrees_of_freedom,
            unavailable_targets,
        })
    }
    fn creation_dimension_structure(
        &self,
        sketch: DocumentSketchId,
        solved: &DocumentSketchSolveOutcome,
        options: SolveOptions,
    ) -> Result<(usize, Vec<SketchError>), SketchError> {
        match solved_sketch_dof(solved, sketch) {
            Ok(dof) => Ok((dof, Vec::new())),
            Err(SketchError::Parameter { .. }) => {
                let anchors = self.resolve_sketch_projected_points(sketch)?;
                let analysis = self
                    .sketches
                    .get(sketch)?
                    .geometry()
                    .analyze_available_targets(
                        &self.parameters,
                        &anchors,
                        options.rank_tolerance,
                    )?;
                Ok((analysis.structure.degrees_of_freedom, analysis.unavailable))
            }
            Err(cause) => Err(cause),
        }
    }
}
