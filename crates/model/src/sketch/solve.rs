// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchConstraintId, SketchError, SketchPoint2D, SketchRelationId, grounded,
    source::source_failure,
};
use crate::parameter::ParameterCollection;
use oblikovati_solve::{
    diagnostics::{ConstraintStructure, analyze_structure, redundant_sources},
    nonlinear::{
        NonlinearResult, PreferenceTermination, SolveError, SolveOptions,
        solve_constraints_with_constants, solve_with_preference,
    },
    residual::evaluate_residuals,
};

/// Immutable solve candidate; callers decide how document health permits adoption.
#[derive(Clone, Debug)]
pub struct SketchSolveOutcome {
    candidate: Sketch2D,
    numerical: NonlinearResult,
    origins: Vec<SketchRelationId>,
    parameter_dependencies: Vec<u64>,
    operands: Vec<Vec<SketchResidualOperand>>,
}
/// Identity and accepted value for a variable read by one prepared relation.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchResidualOperand {
    /// Column in the complete pre-elimination sketch variable layout.
    pub column: usize,
    /// A point coordinate when known; other variable kinds remain explicitly unidentified.
    pub point: Option<(super::SketchEntityId, usize)>,
    /// The accepted binary64 operand, not an uncertainty bound.
    pub value: f64,
    /// True when the solver took this point from the supplied grounded anchors.
    pub externally_grounded: bool,
}
/// One raw solver equation with its relation and convergence domain, not a certified bound.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchResidualRow {
    pub relation: SketchRelationId,
    pub equation: usize,
    /// Source-level inputs; a row may use only a subset. Targets are not included.
    pub operands: Vec<SketchResidualOperand>,
    pub value: f64,
    pub convergence_domain: oblikovati_solve::residual::ResidualTolerance,
    /// Physical unit of the raw value; unknown rows cannot be qualified by unit.
    pub unit: oblikovati_solve::residual::ResidualValueUnit,
}
impl SketchSolveOutcome {
    /// Retain raw source-ordered evidence; example: `outcome.residual_rows()`.
    pub fn residual_rows(&self) -> Vec<SketchResidualRow> {
        let numerical = &self.numerical;
        numerical
            .evaluation
            .source_rows()
            .iter()
            .enumerate()
            .flat_map(|(source, rows)| {
                let relation = self.origins[source];
                rows.clone()
                    .enumerate()
                    .map(move |(equation, index)| SketchResidualRow {
                        relation,
                        equation,
                        operands: self.operands[source].clone(),
                        value: numerical.evaluation.values()[index],
                        convergence_domain: numerical.residual_domains[index],
                        unit: numerical.residual_units[index],
                    })
            })
            .collect()
    }
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
    /// Inspect all unsatisfied authored and structural relations in source order; example: `outcome.conflicting_relations()`.
    pub fn conflicting_relations(&self) -> Vec<SketchRelationId> {
        // #3802: residual severity can change across platforms; report stable source order.
        let mut sources = self.numerical.conflicts.clone();
        sources.sort_unstable();
        sources
            .into_iter()
            .map(|index| self.origins[index])
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
    /// Solve with temporary point targets while preserving authored constraints and actual DOF.
    /// Example: `sketch.solve_with_point_preferences(&parameters, &anchors, &targets, options, || false)?`.
    pub fn solve_with_point_preferences(
        &self,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        targets: &[SketchPoint2D],
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<(SketchSolveOutcome, PreferenceTermination), SketchError> {
        let prepared = grounded::prepare(self, parameters, anchors)?;
        let (preferences, blocked) = self.preferred_coordinates(&prepared, targets)?;
        let result = solve_with_preference(
            &prepared.warm_start_values(self),
            &prepared.blocks(),
            &prepared.constants,
            &preferences,
            options,
            cancelled,
        )
        .map_err(|cause| source_failure(&prepared.origins, cause))?;
        let candidate = self.candidate_values(&prepared.restore(&result.solution.values)?)?;
        let operands = residual_operands(self, &prepared, &result.solution.values, anchors)?;
        let outcome = SketchSolveOutcome {
            operands,
            origins: prepared.origins,
            candidate,
            numerical: result.solution,
            parameter_dependencies: prepared.parameter_dependencies,
        };
        let status = if blocked && result.preference == PreferenceTermination::Converged {
            PreferenceTermination::Unavailable
        } else {
            result.preference
        };
        Ok((outcome, status))
    }

    fn preferred_coordinates(
        &self,
        prepared: &grounded::GroundedSketch,
        targets: &[SketchPoint2D],
    ) -> Result<(Vec<(usize, f64)>, bool), SketchError> {
        let mut seen = std::collections::BTreeSet::new();
        let mut result = Vec::new();
        let mut blocked = false;
        for target in targets {
            if !seen.insert(target.id) || !target.position.iter().all(|value| value.is_finite()) {
                return Err(SketchError::Definition(format!(
                    "preferred point {target:?}: expected a unique point with finite coordinates"
                )));
            }
            let index = self
                .points()
                .iter()
                .position(|point| point.id == target.id)
                .ok_or_else(|| {
                    SketchError::Definition(format!(
                        "preferred point {:?}: expected an owned point",
                        target.id
                    ))
                })?;
            for (axis, value) in target.position.into_iter().enumerate() {
                let (slot, conflict) = prepared.preference_slot(index * 2 + axis, value);
                if let Some(slot) = slot {
                    result.push((slot, value));
                }
                blocked |= conflict;
            }
        }
        Ok((result, blocked))
    }
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
        let warm_values = prepared.warm_start_values(self);
        let numerical = solve_constraints_with_constants(
            &warm_values,
            &prepared.blocks(),
            &prepared.constants,
            options,
            cancelled,
        )
        .map_err(|cause| source_failure(&prepared.origins, cause))?;
        let candidate = self.candidate_values(&prepared.restore(&numerical.values)?)?;
        let operands = residual_operands(self, &prepared, &numerical.values, anchors)?;
        Ok(SketchSolveOutcome {
            operands,
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

#[cfg(test)]
#[path = "../../tests/sketch/conflict_order.rs"]
mod conflict_order_tests;

fn residual_operands(
    sketch: &Sketch2D,
    prepared: &grounded::GroundedSketch,
    values: &[f64],
    anchors: &[SketchPoint2D],
) -> Result<Vec<Vec<SketchResidualOperand>>, SketchError> {
    let restored = prepared.restore(values)?;
    Ok(prepared
        .operand_columns()
        .into_iter()
        .map(|columns| {
            columns
                .into_iter()
                .map(|column| residual_operand(sketch, &restored, anchors, column))
                .collect()
        })
        .collect())
}
fn residual_operand(
    sketch: &Sketch2D,
    values: &[f64],
    anchors: &[SketchPoint2D],
    column: usize,
) -> SketchResidualOperand {
    let point = sketch
        .points()
        .get(column / 2)
        .map(|point| (point.id, column % 2));
    let externally_grounded =
        point.is_some_and(|(id, _)| anchors.iter().any(|anchor| anchor.id == id));
    SketchResidualOperand {
        column,
        point,
        value: values[column],
        externally_grounded,
    }
}

#[cfg(test)]
mod operand_tests {
    use super::*;

    #[test]
    fn non_point_operand_is_explicitly_unidentified() {
        let mut sketch = Sketch2D::default();
        sketch
            .add_point(super::super::SketchEntityId(1), [2.0, -3.0])
            .unwrap();
        let operand = residual_operand(&sketch, &[2.0, -3.0, 4.0], &[], 2);
        assert_eq!(operand.point, None);
        assert_eq!(operand.value, 4.0);
        assert!(!operand.externally_grounded);
    }
}
