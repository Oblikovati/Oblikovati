// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, SketchConstraintReservationError};
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketchSolveOutcome},
    sketch::{
        Sketch2D, SketchConstraint, SketchConstraintId, SketchEntityId, SketchError,
        SketchGeometricConstraintCreation, SketchPoint2D,
    },
};
use oblikovati_solve::nonlinear::{PreferenceTermination, SolveOptions};
use std::{error::Error, fmt};
/// One real authored relation, its geometric row, and resulting current numerical structure.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchGeometricCreationReport {
    pub constraint: SketchConstraint,
    pub geometric_index: usize,
    pub degrees_of_freedom: usize,
    /// The requested temporary preference outcome, or None when no preference was requested.
    pub preference: Option<PreferenceTermination>,
}
/// Geometry failure and permanent identity exhaustion remain separate diagnostic causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchGeometricCreationError {
    Sketch(SketchError),
    Identity(SketchConstraintReservationError),
    Readiness(crate::document_sketch::SketchReadinessError),
}
impl DocumentSnapshot {
    /// Append and evaluate one geometric relation on an isolated owner candidate.
    /// Example: `snapshot.create_sketch_geometric_constraint(sketch, &input, options, ids)?;` inside an edit.
    pub fn create_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchGeometricCreationReport, SketchGeometricCreationError> {
        self.create_geometric_with_preference(sketch, input, None, options, ids)
    }
    /// Create a relation while preferring one point's current position during its initial solve.
    /// Example: `snapshot.create_sketch_geometric_constraint_preferred(sketch, &input, point, options, ids)?`.
    pub fn create_sketch_geometric_constraint_preferred(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        point: SketchEntityId,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchGeometricCreationReport, SketchGeometricCreationError> {
        self.create_geometric_with_preference(sketch, input, Some(point), options, ids)
    }
    fn create_geometric_with_preference(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        preferred: Option<SketchEntityId>,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchGeometricCreationReport, SketchGeometricCreationError> {
        let mut geometry = self
            .prepare_geometric_creation(sketch, input, options)
            .map_err(SketchGeometricCreationError::Sketch)?;
        if preferred.is_some() && !input.requires_current_seed() {
            geometry = self
                .current_sketch_authoring_geometry(sketch, options)
                .map_err(SketchGeometricCreationError::Sketch)?;
        }
        let target = preferred
            .map(|point| geometry.point(point).copied())
            .transpose()
            .map_err(SketchGeometricCreationError::Sketch)?;
        let id = ids
            .allocate_sketch_constraint(sketch)
            .map_err(SketchGeometricCreationError::Identity)?;
        input
            .append(&mut geometry, id)
            .map_err(SketchGeometricCreationError::Sketch)?;
        let preference = self
            .apply_geometric_preference(sketch, &mut geometry, target, options)
            .map_err(SketchGeometricCreationError::Sketch)?;
        let (candidate, mut report) = self
            .evaluate_geometric_creation(sketch, geometry, id, options)
            .map_err(SketchGeometricCreationError::Sketch)?;
        report.preference = preference;
        // #3818: zero DOF does not certify convergence; refuse before adopting the candidate.
        candidate
            .require_converged_sketch(sketch)
            .map_err(SketchGeometricCreationError::Readiness)?;
        self.sketches = candidate.sketches;
        Ok(report)
    }
    fn apply_geometric_preference(
        &self,
        sketch: DocumentSketchId,
        geometry: &mut Sketch2D,
        target: Option<SketchPoint2D>,
        options: SolveOptions,
    ) -> Result<Option<PreferenceTermination>, SketchError> {
        let Some(target) = target else {
            return Ok(None);
        };
        let anchors = self.resolve_sketch_projected_points(sketch)?;
        let (outcome, status) = geometry.solve_with_point_preferences(
            &self.parameters,
            &anchors,
            &[target],
            options,
            || false,
        )?;
        let termination = outcome.numerical().termination;
        if termination != oblikovati_solve::nonlinear::SolveTermination::Converged {
            return Err(SketchError::Definition(format!(
                "preferred sketch constraint solve {termination:?}: expected converged authored constraints"
            )));
        }
        *geometry = outcome.into_candidate();
        Ok(Some(status))
    }
    fn prepare_geometric_creation(
        &self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        options: SolveOptions,
    ) -> Result<Sketch2D, SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        input.validate(geometry)?;
        if input.requires_current_seed() {
            return self.current_sketch_authoring_geometry(sketch, options);
        }
        Ok(geometry.clone())
    }
    fn evaluate_geometric_creation(
        &self,
        sketch: DocumentSketchId,
        geometry: Sketch2D,
        id: SketchConstraintId,
        options: SolveOptions,
    ) -> Result<(DocumentSnapshot, SketchGeometricCreationReport), SketchError> {
        let geometric_index = geometry.geometric_constraint_index(id)?;
        let constraint = created_constraint(&geometry, geometric_index, id)?;
        let mut candidate = self.clone();
        candidate.sketches.replace_geometry(sketch, geometry)?;
        let solved = candidate.solve_one_document_sketch(sketch, options, || false)?;
        let degrees_of_freedom = solved_sketch_dof(&solved, sketch)?;
        let report = SketchGeometricCreationReport {
            constraint,
            geometric_index,
            degrees_of_freedom,
            preference: None,
        };
        candidate.sketches = solved.candidate;
        Ok((candidate, report))
    }
}
pub(super) fn solved_sketch_dof(
    solved: &DocumentSketchSolveOutcome,
    sketch: DocumentSketchId,
) -> Result<usize, SketchError> {
    Ok(solved_sketch_numerical(solved, sketch)?
        .structure
        .degrees_of_freedom)
}
pub(super) fn solved_sketch_numerical(
    solved: &DocumentSketchSolveOutcome,
    sketch: DocumentSketchId,
) -> Result<&oblikovati_solve::nonlinear::NonlinearResult, SketchError> {
    let report = solved
        .report
        .reports
        .iter()
        .find(|report| report.sketch == sketch)
        .ok_or_else(|| {
            SketchError::Definition(format!(
                "solved sketch owner {}: expected its current solve report",
                sketch.0
            ))
        })?;
    let result = report.result.as_ref().map_err(Clone::clone)?;
    Ok(result.numerical())
}
impl SketchGeometricCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Identity(cause) => cause,
            Self::Readiness(cause) => cause,
        }
    }
}
impl fmt::Display for SketchGeometricCreationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(output, "geometric constraint creation: {}", self.cause())
    }
}
impl Error for SketchGeometricCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause())
    }
}

fn created_constraint(
    geometry: &Sketch2D,
    geometric_index: usize,
    id: SketchConstraintId,
) -> Result<SketchConstraint, SketchError> {
    let constraint = geometry
        .geometric_constraints()
        .nth(geometric_index)
        .cloned()
        .ok_or_else(|| {
            SketchError::Definition(format!(
                "created constraint {id:?}: expected its geometric row {geometric_index}"
            ))
        })?;
    Ok(constraint)
}
