// SPDX-License-Identifier: GPL-2.0-only
//! Conditional Krawczyk check on a retained solve, not an operation gate.
mod krawczyk;
use super::{
    SketchIntervalTarget, SketchRelationId, SketchSolveOutcome,
    interval_equations::enclosure::Enclosure,
};
use crate::parameter::ParameterCollection;
use oblikovati_solve::nonlinear::SolveTermination;

/// An explicit point-value assumption for one authored fixed coordinate.
#[derive(Clone, Debug, PartialEq)]
pub struct FixedCoordinateAssumption {
    pub relation: SketchRelationId,
    pub equation: usize,
    pub value: f64,
}
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
    parameter_bounds: Option<Vec<super::ConditionalParameterBound>>,
    fixed_assumptions: Vec<FixedCoordinateAssumption>,
    projected_anchor: Option<(super::SketchPoint2D, [[f64; 2]; 2])>,
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
    /// Read source-derived target bounds, if this check used them; example: `root.parameter_bounds()`.
    pub fn parameter_bounds(&self) -> Option<&[super::ConditionalParameterBound]> {
        self.parameter_bounds.as_deref()
    }
    /// Read explicit fixed-coordinate assumptions; example: `root.fixed_assumptions()`.
    pub fn fixed_assumptions(&self) -> &[FixedCoordinateAssumption] {
        &self.fixed_assumptions
    }
    /// Read the conditional projected-anchor input, when present; example: `root.projected_anchor()`.
    pub fn projected_anchor(&self) -> Option<(super::SketchPoint2D, [[f64; 2]; 2])> {
        self.projected_anchor
    }
    /// Read the verified interval norm upper bound; example: `root.contraction_upper()`.
    pub fn contraction_upper(&self) -> f64 {
        self.contraction_upper
    }
}

impl SketchSolveOutcome {
    /// Check with bounds derived from supported authored parameter sources.
    /// Example: `outcome.conditional_root_from_sources(&parameters, &box_values, &fixed, &inverse)?`.
    /// Source bounds and interval arithmetic remain conditional; this does not qualify a feature.
    pub fn conditional_root_from_sources(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        fixed: &[FixedCoordinateAssumption],
        preconditioner: &[Vec<f64>],
    ) -> Result<ConditionalSketchRoot, String> {
        if !self.source_stamp_current(parameters) {
            return Err(
                "parameter source graph: expected the unchanged accepted solve inputs".into(),
            );
        }
        let (targets, evidence) = source_targets(self, parameters)?;
        let mut root =
            self.conditional_root(parameters, bounds, &targets, fixed, preconditioner)?;
        root.parameter_bounds = Some(evidence);
        Ok(root)
    }
    /// Test a retained solve against a finite box and caller-supplied preconditioner.
    /// Example: `outcome.conditional_root(&parameters, &box_values, &targets, &fixed, &inverse)?`.
    /// This is conditional evidence only; it does not qualify a feature or validate input uncertainty.
    pub fn conditional_root(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        targets: &[SketchIntervalTarget],
        fixed: &[FixedCoordinateAssumption],
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
        verify_fixed_assumptions(self, fixed)?;
        let point_box: Vec<_> = center.iter().map(|value| [*value, *value]).collect();
        let point_rows = self
            .candidate()
            .interval_equations_with_targets(parameters, &point_box, targets)
            .map_err(|e| e.to_string())?;
        let box_input: Vec<_> = box_rows.iter().map(krawczyk::RootRow::from).collect();
        let point_input: Vec<_> = point_rows.iter().map(krawczyk::RootRow::from).collect();
        let result = krawczyk::evaluate(&center, bounds, &box_input, &point_input, preconditioner)?;
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
            parameter_bounds: None,
            fixed_assumptions: fixed.to_vec(),
            projected_anchor: None,
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

fn source_targets(
    outcome: &SketchSolveOutcome,
    parameters: &ParameterCollection<u64>,
) -> Result<
    (
        Vec<SketchIntervalTarget>,
        Vec<super::ConditionalParameterBound>,
    ),
    String,
> {
    let mut targets = Vec::new();
    let mut evidence = Vec::new();
    for row in outcome.residual_rows() {
        let row_targets = row
            .targets
            .ok_or_else(|| format!("relation {:?}: expected attributed targets", row.relation))?;
        for input in row_targets {
            let super::SketchTargetSource::Parameter(id) = input.source else {
                if matches!(input.source, super::SketchTargetSource::Unknown) {
                    return Err(format!(
                        "relation {:?}: expected a known target source",
                        row.relation
                    ));
                }
                continue;
            };
            if targets.iter().any(|target: &SketchIntervalTarget| {
                target.relation == row.relation && target.parameter == id
            }) {
                continue;
            }
            let bound = super::conditional_parameter_bound(parameters, id)?;
            targets.push(SketchIntervalTarget {
                relation: row.relation,
                parameter: id,
                bounds: bound.bounds(),
            });
            evidence.push(bound);
        }
    }
    Ok((targets, evidence))
}

fn verify_fixed_assumptions(
    outcome: &SketchSolveOutcome,
    assumptions: &[FixedCoordinateAssumption],
) -> Result<(), String> {
    let mut used = vec![false; assumptions.len()];
    for row in outcome.residual_rows() {
        let Some(targets) = row.targets else {
            continue;
        };
        for target in targets {
            if !matches!(target.source, super::SketchTargetSource::AuthoredPosition) {
                continue;
            }
            let matches: Vec<_> = assumptions
                .iter()
                .enumerate()
                .filter(|(_, value)| {
                    value.relation == row.relation
                        && value.equation == row.equation
                        && value.value.to_bits() == target.value.to_bits()
                })
                .collect();
            let [(index, _)] = matches.as_slice() else {
                return Err(format!(
                    "fixed target {:?} row {} value {:?}: expected one explicit exact-value assumption",
                    row.relation, row.equation, target.value
                ));
            };
            used[*index] = true;
        }
    }
    if used.iter().any(|used| !used) {
        return Err(format!(
            "fixed assumptions {assumptions:?}: expected each assumption to match one fixed coordinate"
        ));
    }
    Ok(())
}

impl SketchSolveOutcome {
    pub(crate) fn conditional_root_with_origin(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        fixed: &[FixedCoordinateAssumption],
        preconditioner: &[Vec<f64>],
        anchor: super::SketchPoint2D,
        origin_bounds: [[f64; 2]; 2],
    ) -> Result<ConditionalSketchRoot, String> {
        if self.numerical().termination != SolveTermination::Converged
            || !self.source_stamp_current(parameters)
        {
            return Err(
                "projected root: expected a converged solve with unchanged parameter sources"
                    .into(),
            );
        }
        let (center, columns) = self.verified_projected_center(parameters, anchor)?;
        validate_box(&center, bounds)?;
        for axis in 0..2 {
            let interval = Enclosure::new(origin_bounds[axis])?.bounds();
            if interval[0] < bounds[columns[axis]][0] || interval[1] > bounds[columns[axis]][1] {
                return Err(format!(
                    "origin anchor {anchor:?} bounds {interval:?}: expected the entire anchor bound inside its variable box"
                ));
            }
        }
        verify_fixed_assumptions(self, fixed)?;
        let (targets, evidence) = source_targets(self, parameters)?;
        let box_rows = self
            .candidate()
            .interval_equations_with_targets(parameters, bounds, &targets)
            .map_err(|e| e.to_string())?;
        let active = self.numerical().values.len();
        if box_rows.len() != active
            || self.numerical().evaluation.values().len() != active
            || center.len() != active + 2
        {
            return Err(format!(
                "projected system: expected {active} original equations and two eliminated coordinates, got {} equations and {} variables",
                box_rows.len(),
                center.len()
            ));
        }
        let point_box: Vec<_> = center.iter().map(|value| [*value, *value]).collect();
        let point_rows = self
            .candidate()
            .interval_equations_with_targets(parameters, &point_box, &targets)
            .map_err(|e| e.to_string())?;
        let mut box_input: Vec<_> = box_rows.iter().map(krawczyk::RootRow::from).collect();
        let mut point_input: Vec<_> = point_rows.iter().map(krawczyk::RootRow::from).collect();
        for axis in 0..2 {
            box_input.push(anchor_equation(
                bounds[columns[axis]],
                origin_bounds[axis],
                columns[axis],
                center.len(),
            )?);
            point_input.push(anchor_equation(
                point_box[columns[axis]],
                origin_bounds[axis],
                columns[axis],
                center.len(),
            )?);
        }
        let result = krawczyk::evaluate(&center, bounds, &box_input, &point_input, preconditioner)?;
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
            targets,
            parameter_bounds: Some(evidence),
            fixed_assumptions: fixed.to_vec(),
            projected_anchor: Some((anchor, origin_bounds)),
            contraction_upper: result.contraction_upper,
        })
    }
}
fn anchor_equation(
    variable: [f64; 2],
    source: [f64; 2],
    column: usize,
    count: usize,
) -> Result<krawczyk::RootRow, String> {
    let value = Enclosure::new(variable)?
        .difference(Enclosure::new(source)?)?
        .bounds();
    let mut jacobian = vec![[0., 0.]; count];
    jacobian[column] = [1., 1.];
    Ok(krawczyk::RootRow { value, jacobian })
}

#[cfg(test)]
mod projected_equation_tests {
    use super::*;
    #[test]
    fn anchor_equation_contains_input_uncertainty_and_identity_jacobian() {
        let row = anchor_equation([-0.2, 0.4], [-0.1, 0.3], 1, 3).unwrap();
        let q = |value| num_rational::BigRational::from_float(value).unwrap();
        assert!(q(row.value[0]) <= q(-0.2) - q(0.3));
        assert!(q(row.value[1]) >= q(0.4) - q(-0.1));
        assert_eq!(row.jacobian, [[0., 0.], [1., 1.], [0., 0.]]);
    }
}
