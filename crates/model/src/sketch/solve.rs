// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchError, compile};
use crate::parameter::ParameterCollection;
use oblikovati_solve::{
    diagnostics::{ConstraintStructure, analyze_structure, redundant_sources},
    nonlinear::{NonlinearResult, SolveError, SolveOptions, solve_constraints},
    residual::{EvaluationError, evaluate_residuals},
};

/// Immutable solve candidate; callers decide how document health permits adoption.
#[derive(Clone, Debug)]
pub struct SketchSolveOutcome {
    candidate: Sketch2D,
    numerical: NonlinearResult,
    parameter_dependencies: Vec<u64>,
}
impl SketchSolveOutcome {
    /// Inspect accepted numerical values without changing the input sketch; example: `outcome.candidate()`.
    pub fn candidate(&self) -> &Sketch2D {
        &self.candidate
    }
    /// Inspect convergence separately from DOF; example: `outcome.numerical().termination`.
    pub fn numerical(&self) -> &NonlinearResult {
        &self.numerical
    }
    /// Read directly referenced dimension parameters in first-use order; example: `outcome.parameter_dependencies()`.
    pub fn parameter_dependencies(&self) -> &[u64] {
        &self.parameter_dependencies
    }
    /// Resolve unsatisfied source indices to authored identities; example: `outcome.conflicting_constraints()`.
    pub fn conflicting_constraints(&self) -> Vec<SketchConstraintId> {
        self.numerical
            .conflicts
            .iter()
            .map(|&index| self.candidate.constraints[index].id)
            .collect()
    }
    /// Inspect alternative individual removals using the captured Jacobian; example: `outcome.redundant_constraints(1e-9)?`.
    pub fn redundant_constraints(
        &self,
        tolerance: f64,
    ) -> Result<Vec<SketchConstraintId>, SketchError> {
        Ok(redundant_sources(&self.numerical.evaluation, tolerance)
            .map_err(SolveError::from)?
            .into_iter()
            .map(|index| self.candidate.constraints[index].id)
            .collect())
    }
    /// Take the candidate for an explicit document edit; example: `let candidate = outcome.into_candidate()`.
    pub fn into_candidate(self) -> Sketch2D {
        self.candidate
    }
}
impl Sketch2D {
    /// Solve on owned values with healthy parameter targets; example: `sketch.solve(&parameters, SolveOptions::default(), || false)?`.
    pub fn solve(
        &self,
        parameters: &ParameterCollection<u64>,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<SketchSolveOutcome, SketchError> {
        let prepared = compile::prepare(self, parameters)?;
        let numerical = solve_constraints(&prepared.values, &prepared.blocks(), options, cancelled)
            .map_err(|cause| self.constraint_failure(cause))?;
        let mut candidate = self.clone();
        for (point, values) in candidate
            .points
            .iter_mut()
            .zip(numerical.values.as_chunks::<2>().0.iter())
        {
            point.position = [values[0], values[1]];
        }
        Ok(SketchSolveOutcome {
            candidate,
            numerical,
            parameter_dependencies: prepared.parameter_dependencies,
        })
    }
    /// Inspect local DOF without changing geometry or iterating; example: `sketch.analyze(&parameters, 1e-9)?`.
    pub fn analyze(
        &self,
        parameters: &ParameterCollection<u64>,
        rank_tolerance: f64,
    ) -> Result<ConstraintStructure, SketchError> {
        let prepared = compile::prepare(self, parameters)?;
        let evaluation = evaluate_residuals(&prepared.values, &prepared.blocks())
            .map_err(|cause| self.constraint_failure(cause.into()))?;
        Ok(analyze_structure(&evaluation, rank_tolerance).map_err(SolveError::from)?)
    }
    fn constraint_failure(&self, cause: SolveError) -> SketchError {
        if let SolveError::Evaluation(EvaluationError::Source { block, .. }) = &cause
            && let Some(constraint) = self.constraints.get(*block)
            && let SolveError::Evaluation(cause) = cause
        {
            return SketchError::Residual {
                constraint: constraint.id,
                cause,
            };
        }
        cause.into()
    }
}
