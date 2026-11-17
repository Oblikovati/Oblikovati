// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, sketch_relation_create::solved_sketch_numerical,
};
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketchSolveOutcome},
    quantity::Quantity,
    sketch::{Sketch2D, SketchConstraintId, SketchDimensionalConstraintCreation, SketchError},
};
use oblikovati_solve::nonlinear::SolveOptions;
mod error;
mod role;
pub use error::SketchDimensionCreationError;
use role::DimensionRolePolicy;
type AuthoringDatums<'a> = Option<&'a crate::feature::work_geometry::WorkGeometryEvaluation>;
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
        self.create_dimension_with_role(
            sketch,
            input,
            options,
            ids,
            DimensionRolePolicy::Explicit,
            None,
        )
    }
    fn create_dimension_with_role(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        policy: DimensionRolePolicy,
        datums: AuthoringDatums<'_>,
    ) -> Result<SketchDimensionCreationReport, SketchDimensionCreationError> {
        let geometry = self.prepare_dimension_creation(sketch, input, options, datums)?;
        let prior_redundancy =
            self.dimension_prior_redundancy(sketch, &geometry, options, datums)?;
        let probe =
            self.prepare_dimension_role(sketch, &geometry, input, policy, options, datums)?;
        let reserved = self.reserve_dimension_creation(sketch, ids)?;
        let mut candidate = self.clone();
        candidate.append_owned_dimension(sketch, input, geometry, &reserved)?;
        candidate.classify_dimension_role(sketch, reserved.constraint, probe, options)?;
        let report = candidate.evaluate_created_dimension(
            sketch,
            &reserved,
            options,
            datums,
            prior_redundancy,
        )?;
        candidate.confirm_dimension_creation(sketch, &report, datums)?;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(report)
    }
    fn dimension_prior_redundancy(
        &self,
        sketch: DocumentSketchId,
        geometry: &Sketch2D,
        options: SolveOptions,
        datums: AuthoringDatums<'_>,
    ) -> Result<usize, SketchError> {
        let anchors = self.sketch_authoring_anchors(sketch, datums)?;
        Ok(geometry
            .analyze_with_grounded_points(&self.parameters, &anchors, options.rank_tolerance)?
            .redundant_equations)
    }
    fn confirm_dimension_creation(
        &self,
        sketch: DocumentSketchId,
        report: &SketchDimensionCreationReport,
        datums: AuthoringDatums<'_>,
    ) -> Result<(), SketchDimensionCreationError> {
        // #3824: DOF cannot certify a numerical result with available targets.
        if report.unavailable_targets.is_empty() {
            self.require_converged_sketch_from(sketch, datums)
                .map_err(SketchDimensionCreationError::Readiness)?;
        }
        Ok(())
    }
    fn prepare_dimension_creation(
        &self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        options: SolveOptions,
        datums: AuthoringDatums<'_>,
    ) -> Result<Sketch2D, SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        input.operands.validate(geometry)?;
        if let Some(position) = input.text_point {
            crate::sketch::require_position(position)?;
        }
        if input.operands.requires_current_seed() {
            return self.current_sketch_authoring_geometry_from(sketch, options, datums);
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
        datums: AuthoringDatums<'_>,
        prior_redundancy: usize,
    ) -> Result<SketchDimensionCreationReport, SketchError> {
        let solved = match datums {
            Some(datums) => {
                self.solve_one_document_sketch_with_datums(sketch, options, || false, datums)?
            }
            None => self.solve_one_document_sketch(sketch, options, || false)?,
        };
        let (degrees_of_freedom, unavailable_targets) =
            self.creation_dimension_structure(sketch, &solved, options, datums, prior_redundancy)?;
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
        datums: AuthoringDatums<'_>,
        prior_redundancy: usize,
    ) -> Result<(usize, Vec<SketchError>), SketchError> {
        match solved_sketch_numerical(solved, sketch) {
            Ok(numerical) => self.available_dimension_structure(numerical, prior_redundancy),
            Err(SketchError::Parameter { .. }) => {
                self.unavailable_dimension_structure(sketch, options, datums)
            }
            Err(cause) => Err(cause),
        }
    }
    fn available_dimension_structure(
        &self,
        numerical: &oblikovati_solve::nonlinear::NonlinearResult,
        prior_redundancy: usize,
    ) -> Result<(usize, Vec<SketchError>), SketchError> {
        use oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;
        let structure = &numerical.structure;
        let explicit_driving = self.effective_sketch_settings().over_constrained_behavior
            == OVER_CONSTRAINED_APPLY_DRIVING;
        if numerical.termination == oblikovati_solve::nonlinear::SolveTermination::Converged
            && structure.redundant_equations > prior_redundancy
            && !explicit_driving
        {
            return Err(SketchError::Definition(format!(
                "dimension candidate has {} redundant equations versus {prior_redundancy} before creation: expected no increase; use a reference dimension for a measured relation",
                structure.redundant_equations
            )));
        }
        Ok((structure.degrees_of_freedom, Vec::new()))
    }
    fn unavailable_dimension_structure(
        &self,
        sketch: DocumentSketchId,
        options: SolveOptions,
        datums: AuthoringDatums<'_>,
    ) -> Result<(usize, Vec<SketchError>), SketchError> {
        let anchors = self.sketch_authoring_anchors(sketch, datums)?;
        let analysis = self
            .sketches
            .get(sketch)?
            .geometry()
            .analyze_available_targets(&self.parameters, &anchors, options.rank_tolerance)?;
        Ok((analysis.structure.degrees_of_freedom, analysis.unavailable))
    }
}
