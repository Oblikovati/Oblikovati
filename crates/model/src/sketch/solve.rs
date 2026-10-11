// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchConstraintId, SketchError, SketchPoint2D, SketchRelationId, grounded,
    source::source_failure,
};
use crate::parameter::ParameterCollection;
use oblikovati_solve::{
    diagnostics::{ConstraintStructure, analyze_structure, redundant_sources},
    nonlinear::{NonlinearResult, SolveError, SolveOptions, solve_constraints},
    residual::evaluate_residuals,
};

/// Immutable solve candidate; callers decide how document health permits adoption.
#[derive(Clone, Debug)]
pub struct SketchSolveOutcome {
    candidate: Sketch2D,
    numerical: NonlinearResult,
    origins: Vec<SketchRelationId>,
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
    /// Inspect only authored conflicts; use `conflicting_relations()` for structural sources; example: `outcome.conflicting_constraints()`.
    pub fn conflicting_constraints(&self) -> Vec<SketchConstraintId> {
        self.conflicting_relations()
            .into_iter()
            .filter_map(SketchRelationId::authored)
            .collect()
    }
    /// Inspect only authored redundancies; use `redundant_relations()` for structural sources; example: `outcome.redundant_constraints(1e-9)?`.
    pub fn redundant_constraints(
        &self,
        tolerance: f64,
    ) -> Result<Vec<SketchConstraintId>, SketchError> {
        Ok(self
            .redundant_relations(tolerance)?
            .into_iter()
            .filter_map(SketchRelationId::authored)
            .collect())
    }
    /// Inspect all unsatisfied authored and structural relations; example: `outcome.conflicting_relations()`.
    pub fn conflicting_relations(&self) -> Vec<SketchRelationId> {
        self.numerical
            .conflicts
            .iter()
            .map(|&index| self.origins[index])
            .collect()
    }
    /// Inspect all individually redundant sources, including structural equations; example: `outcome.redundant_relations(1e-9)?`.
    pub fn redundant_relations(
        &self,
        tolerance: f64,
    ) -> Result<Vec<SketchRelationId>, SketchError> {
        Ok(redundant_sources(&self.numerical.evaluation, tolerance)
            .map_err(SolveError::from)?
            .into_iter()
            .map(|index| self.origins[index])
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
        self.solve_with_grounded_points(parameters, &[], options, cancelled)
    }
    /// Treat current reference coordinates as constants through the shared solver.
    /// Example: `sketch.solve_with_grounded_points(&parameters, &anchors, options, || false)?`.
    pub fn solve_with_grounded_points(
        &self,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<SketchSolveOutcome, SketchError> {
        let prepared = grounded::prepare(self, parameters, anchors)?;
        let numerical = solve_constraints(&prepared.values, &prepared.blocks(), options, cancelled)
            .map_err(|cause| source_failure(&prepared.origins, cause))?;
        let candidate = self.candidate_values(&prepared.restore(&numerical.values)?)?;
        Ok(SketchSolveOutcome {
            origins: prepared.origins,
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
        self.analyze_with_grounded_points(parameters, &[], rank_tolerance)
    }
    /// Inspect active DOF while keeping current reference coordinates exact.
    /// Example: `sketch.analyze_with_grounded_points(&parameters, &anchors, 1e-9)?`.
    pub fn analyze_with_grounded_points(
        &self,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        rank_tolerance: f64,
    ) -> Result<ConstraintStructure, SketchError> {
        let prepared = grounded::prepare(self, parameters, anchors)?;
        let evaluation = evaluate_residuals(&prepared.values, &prepared.blocks())
            .map_err(|cause| source_failure(&prepared.origins, cause.into()))?;
        Ok(analyze_structure(&evaluation, rank_tolerance).map_err(SolveError::from)?)
    }
}
