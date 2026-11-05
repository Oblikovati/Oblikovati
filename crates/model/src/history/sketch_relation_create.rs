// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, SketchConstraintReservationError};
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketchSolveOutcome},
    sketch::{
        Sketch2D, SketchConstraint, SketchConstraintId, SketchError,
        SketchGeometricConstraintCreation,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;
use std::{error::Error, fmt};
/// One real authored relation, its geometric row, and resulting current numerical structure.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchGeometricCreationReport {
    pub constraint: SketchConstraint,
    pub geometric_index: usize,
    pub degrees_of_freedom: usize,
}
/// Geometry failure and permanent identity exhaustion remain separate diagnostic causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchGeometricCreationError {
    Sketch(SketchError),
    Identity(SketchConstraintReservationError),
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
        let mut geometry = self
            .prepare_geometric_creation(sketch, input, options)
            .map_err(SketchGeometricCreationError::Sketch)?;
        let id = ids
            .allocate_sketch_constraint(sketch)
            .map_err(SketchGeometricCreationError::Identity)?;
        input
            .append(&mut geometry, id)
            .map_err(SketchGeometricCreationError::Sketch)?;
        let (sketches, report) = self
            .evaluate_geometric_creation(sketch, geometry, id, options)
            .map_err(SketchGeometricCreationError::Sketch)?;
        self.sketches = sketches;
        Ok(report)
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
    ) -> Result<
        (
            crate::document_sketch::DocumentSketches,
            SketchGeometricCreationReport,
        ),
        SketchError,
    > {
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
        };
        Ok((solved.candidate, report))
    }
}
pub(super) fn solved_sketch_dof(
    solved: &DocumentSketchSolveOutcome,
    sketch: DocumentSketchId,
) -> Result<usize, SketchError> {
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
    Ok(result.numerical().structure.degrees_of_freedom)
}
impl SketchGeometricCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Identity(cause) => cause,
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
