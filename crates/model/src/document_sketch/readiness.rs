// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSketchId, PlacedSketch, PlacedSketchHealth, SketchEvaluation, SketchSolveDiagnostics,
};
use crate::{parameter::ParameterCollection, sketch::SketchError};
use oblikovati_solve::nonlinear::SolveTermination;
use std::{error::Error, fmt};
/// Named reasons that prevent a fully constrained sketch consumer from reading geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchReadinessCause {
    Unevaluated,
    Stale(Vec<u64>),
    ProjectionInputsRequired,
    ProjectionInputsChanged,
    Failed(SketchError),
    Unconverged(SolveTermination),
    FreeVariables(usize),
}
/// Local decline names the sketch and keeps the underlying numerical or parameter failure.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchReadinessError {
    pub sketch: DocumentSketchId,
    pub cause: SketchReadinessCause,
}
impl PlacedSketch {
    /// Refuse stale, unsolved or free geometry before feature consumption; example: `placed.require_fully_constrained(&parameters)?`.
    pub fn require_fully_constrained(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<&SketchSolveDiagnostics, SketchReadinessError> {
        self.require_health(self.health(parameters))
    }
    pub(crate) fn require_fully_constrained_with_projection_inputs(
        &self,
        parameters: &ParameterCollection<u64>,
        projections: &Result<Vec<crate::sketch::SketchPoint2D>, SketchError>,
    ) -> Result<&SketchSolveDiagnostics, SketchReadinessError> {
        self.require_health(self.health_with_projection_inputs(parameters, projections))
    }
    fn require_health<'a>(
        &'a self,
        health: PlacedSketchHealth<'a>,
    ) -> Result<&'a SketchSolveDiagnostics, SketchReadinessError> {
        ready(health).map_err(|cause| SketchReadinessError {
            sketch: self.id,
            cause,
        })
    }
}
fn ready(health: PlacedSketchHealth<'_>) -> Result<&SketchSolveDiagnostics, SketchReadinessCause> {
    match health {
        PlacedSketchHealth::Unevaluated => Err(SketchReadinessCause::Unevaluated),
        PlacedSketchHealth::Stale(ids) => Err(SketchReadinessCause::Stale(ids)),
        PlacedSketchHealth::ProjectionInputsRequired => {
            Err(SketchReadinessCause::ProjectionInputsRequired)
        }
        PlacedSketchHealth::ProjectionInputsChanged => {
            Err(SketchReadinessCause::ProjectionInputsChanged)
        }
        PlacedSketchHealth::Current(SketchEvaluation::Failed(cause)) => {
            Err(SketchReadinessCause::Failed(cause.clone()))
        }
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(diag)) => ready_numerical(diag),
    }
}
fn ready_numerical(
    diag: &SketchSolveDiagnostics,
) -> Result<&SketchSolveDiagnostics, SketchReadinessCause> {
    if diag.termination != SolveTermination::Converged {
        return Err(SketchReadinessCause::Unconverged(diag.termination));
    }
    if diag.structure.degrees_of_freedom != 0 {
        return Err(SketchReadinessCause::FreeVariables(
            diag.structure.degrees_of_freedom,
        ));
    }
    Ok(diag)
}
impl fmt::Display for SketchReadinessCause {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unevaluated => write!(output, "verification absent: expected a current solve"),
            Self::ProjectionInputsRequired => write!(
                output,
                "projected point sources: expected current document resolution"
            ),
            Self::ProjectionInputsChanged => write!(
                output,
                "projected point inputs changed: expected a current solve"
            ),
            Self::Stale(ids) => write_stale_dimensions(output, ids),
            Self::Failed(cause) => write_failed_verification(output, cause),
            Self::Unconverged(stop) => {
                write!(output, "solve termination {stop:?}: expected Converged")
            }
            Self::FreeVariables(count) => write!(output, "degrees of freedom {count}: expected 0"),
        }
    }
}
fn write_stale_dimensions(output: &mut fmt::Formatter<'_>, ids: &[u64]) -> fmt::Result {
    write!(
        output,
        "dimension parameters {ids:?} changed: expected a current solve"
    )
}
fn write_failed_verification(output: &mut fmt::Formatter<'_>, cause: &SketchError) -> fmt::Result {
    write!(
        output,
        "verification failed ({cause}): expected a successful solve"
    )
}
impl fmt::Display for SketchReadinessError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(output, "document sketch {:?}: {}", self.sketch, self.cause)
    }
}
impl Error for SketchReadinessError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match &self.cause {
            SketchReadinessCause::Failed(cause) => Some(cause),
            _ => None,
        }
    }
}
