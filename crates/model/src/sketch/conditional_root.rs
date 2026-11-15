// SPDX-License-Identifier: GPL-2.0-only
//! Conditional Krawczyk check on a retained solve, not an operation gate.
mod krawczyk;
use super::{
    SketchIntervalTarget, SketchRelationId, SketchSolveOutcome,
    interval_equations::enclosure::Enclosure,
};
use crate::parameter::ParameterCollection;
use oblikovati_solve::nonlinear::SolveTermination;

/// Conditional root enclosure, assuming the supplied target bounds and interval backend are valid.
#[derive(Clone, Debug, PartialEq)]
pub struct ConditionalSketchRoot {
    sketch: super::Sketch2D,
    center: Vec<f64>,
    input_box: Vec<[f64; 2]>,
    enclosure: Vec<[f64; 2]>,
    relations: Vec<(SketchRelationId, usize)>,
    /// Retained solve evidence supplies operand and equation identities.
    source_rows: Vec<super::SketchResidualRow>,
    preconditioner: Vec<Vec<f64>>,
    targets: Vec<SketchIntervalTarget>,
    contraction_upper: f64,
}
impl ConditionalSketchRoot {
    /// Read the retained accepted sketch; example: `root.sketch()`.
    pub fn sketch(&self) -> &super::Sketch2D {
        &self.sketch
    }
    /// Read the accepted center; example: `root.center()`.
    pub fn center(&self) -> &[f64] {
        &self.center
    }
    /// Read the assumed input box; example: `root.input_box()`.
    pub fn input_box(&self) -> &[[f64; 2]] {
        &self.input_box
    }
    /// Read the conditional root bounds; example: `root.enclosure()`.
    pub fn enclosure(&self) -> &[[f64; 2]] {
        &self.enclosure
    }
    /// Read equation identities; example: `root.relations()`.
    pub fn relations(&self) -> &[(SketchRelationId, usize)] {
        &self.relations
    }
    /// Read the accepted solve evidence; example: `root.source_rows()`.
    pub fn source_rows(&self) -> &[super::SketchResidualRow] {
        &self.source_rows
    }
    /// Read the checked preconditioner; example: `root.preconditioner()`.
    pub fn preconditioner(&self) -> &[Vec<f64>] {
        &self.preconditioner
    }
    /// Read caller-supplied target assumptions; example: `root.targets()`.
    pub fn targets(&self) -> &[SketchIntervalTarget] {
        &self.targets
    }
    /// Read the verified interval norm upper bound; example: `root.contraction_upper()`.
    pub fn contraction_upper(&self) -> f64 {
        self.contraction_upper
    }
}

impl SketchSolveOutcome {
    /// Test a retained solve against a finite box and caller-supplied preconditioner.
    /// Example: `outcome.conditional_root(&parameters, &box_values, &targets, &inverse)?`.
    /// This is conditional evidence only; it does not qualify a feature or validate input uncertainty.
    pub fn conditional_root(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        targets: &[SketchIntervalTarget],
        preconditioner: &[Vec<f64>],
    ) -> Result<ConditionalSketchRoot, String> {
        if self.numerical().termination != SolveTermination::Converged {
            return Err(format!(
                "solve {:?}: expected converged solve",
                self.numerical().termination
            ));
        }
        let center = self.verified_interval_center(parameters)?;
        validate_box(&center, bounds)?;
        let box_rows = self
            .candidate()
            .interval_equations_with_targets(parameters, bounds, targets)
            .map_err(|e| e.to_string())?;
        if center.is_empty() || box_rows.len() != center.len() {
            return Err(format!(
                "interval system has {} equations and {} variables: expected a nonempty square system",
                box_rows.len(),
                center.len()
            ));
        }
        let point_box: Vec<_> = center.iter().map(|value| [*value, *value]).collect();
        let point_rows = self
            .candidate()
            .interval_equations_with_targets(parameters, &point_box, targets)
            .map_err(|e| e.to_string())?;
        let result = krawczyk::evaluate(&center, bounds, &box_rows, &point_rows, preconditioner)?;
        Ok(ConditionalSketchRoot {
            sketch: self.candidate().clone(),
            center,
            input_box: bounds.to_vec(),
            enclosure: result.enclosure,
            relations: box_rows
                .iter()
                .map(|row| (row.relation, row.equation))
                .collect(),
            source_rows: self.residual_rows(),
            preconditioner: preconditioner.to_vec(),
            targets: targets.to_vec(),
            contraction_upper: result.contraction_upper,
        })
    }
}
fn validate_box(center: &[f64], bounds: &[[f64; 2]]) -> Result<(), String> {
    if center.len() != bounds.len() {
        return Err(format!(
            "interval box length {}: expected {} solve variables",
            bounds.len(),
            center.len()
        ));
    }
    for (index, (&value, &interval)) in center.iter().zip(bounds).enumerate() {
        Enclosure::new(interval)?;
        if value < interval[0] || value > interval[1] {
            return Err(format!(
                "solve variable {index} value {value:?} and box {interval:?}: expected the accepted value inside the box"
            ));
        }
    }
    Ok(())
}
