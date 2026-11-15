// SPDX-License-Identifier: GPL-2.0-only
//! Conditional equation enclosures; these results do not certify a solve.
mod dual;
mod enclosure;
use self::{dual::IntervalDual, enclosure::Enclosure};
use super::{
    DistanceMeasure, Sketch2D, SketchError, SketchRelationId, compile, linear_equations,
    linear_equations::DifferenceScalar,
    residual::{PlanarEquation, PreparedConstraint},
};
use crate::parameter::ParameterCollection;
use std::{error::Error, fmt};

/// Enclosure of one prepared equation and its derivatives over the caller's input box.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchIntervalEquation {
    pub relation: SketchRelationId,
    pub equation: usize,
    pub value: [f64; 2],
    /// One derivative interval per complete pre-elimination sketch variable.
    pub jacobian: Vec<[f64; 2]>,
}
/// A caller-supplied enclosure of one prepared dimensional target.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchIntervalTarget {
    pub relation: SketchRelationId,
    pub parameter: u64,
    pub bounds: [f64; 2],
}
/// An interval input, unsupported source, or enclosure that cannot be evaluated.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchIntervalError {
    Input(String),
    Preparation(SketchError),
    Unsupported(SketchRelationId),
    Arithmetic {
        relation: SketchRelationId,
        cause: String,
    },
}
impl fmt::Display for SketchIntervalError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Input(cause) => write!(f, "{cause}"),
            Self::Preparation(cause) => write!(f, "interval sketch preparation: {cause}"),
            Self::Unsupported(relation) => write!(
                f,
                "interval relation {relation:?}: expected a supported interval equation"
            ),
            Self::Arithmetic { relation, cause } => {
                write!(f, "interval relation {relation:?}: {cause}")
            }
        }
    }
}
impl Error for SketchIntervalError {}
impl Sketch2D {
    /// Enclose supported prepared equations over finite boxes in full solver-variable order.
    /// Example: `sketch.interval_equations(&parameters, &[[0.0, 1.0], [2.0, 3.0]])?`.
    /// These enclosures do not prove existence, uniqueness, input uncertainty, or precision.
    pub fn interval_equations(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
    ) -> Result<Vec<SketchIntervalEquation>, SketchIntervalError> {
        self.interval_equations_with_targets(parameters, bounds, &[])
    }
    /// Evaluate with caller-supplied target bounds tied to their relation and parameter.
    /// Example: `sketch.interval_equations_with_targets(&parameters, &box_values, &targets)?`.
    /// The caller must establish that these bounds enclose the true target inputs.
    pub fn interval_equations_with_targets(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        targets: &[SketchIntervalTarget],
    ) -> Result<Vec<SketchIntervalEquation>, SketchIntervalError> {
        let prepared =
            compile::prepare(self, parameters).map_err(SketchIntervalError::Preparation)?;
        if bounds.len() != prepared.values.len() {
            return Err(SketchIntervalError::Input(format!(
                "interval box has {} variables: expected {}",
                bounds.len(),
                prepared.values.len()
            )));
        }
        let variables = seed(bounds)?;
        let mut rows = Vec::new();
        let mut used = vec![false; targets.len()];
        for (source, relation) in prepared.sources.iter().zip(prepared.origins) {
            rows.extend(source_rows(
                self, source, relation, &variables, targets, &mut used,
            )?);
        }
        if used.iter().any(|used| !used) {
            return Err(SketchIntervalError::Input(format!(
                "interval targets {targets:?}: expected each target to match one supported relation"
            )));
        }
        Ok(rows)
    }
}
fn seed(bounds: &[[f64; 2]]) -> Result<Vec<IntervalDual>, SketchIntervalError> {
    bounds
        .iter()
        .enumerate()
        .map(|(index, &endpoints)| {
            let value = Enclosure::new(endpoints).map_err(SketchIntervalError::Input)?;
            IntervalDual::variable(value, index, bounds.len()).map_err(SketchIntervalError::Input)
        })
        .collect()
}

fn source_rows(
    sketch: &Sketch2D,
    source: &PreparedConstraint,
    relation: SketchRelationId,
    variables: &[IntervalDual],
    targets: &[SketchIntervalTarget],
    used: &mut [bool],
) -> Result<Vec<SketchIntervalEquation>, SketchIntervalError> {
    if source.projection.is_some() {
        return Err(SketchIntervalError::Unsupported(relation));
    }
    let local: Vec<_> = source
        .variables
        .iter()
        .map(|&column| variables[column].clone())
        .collect();
    let values = match source.equation {
        PlanarEquation::Coincident => linear_equations::coincident(&local),
        PlanarEquation::Coordinate(axis) => linear_equations::coordinate(&local, axis),
        PlanarEquation::Fix(position) => fixed_rows(&local, &[position]),
        PlanarEquation::Ground(ref positions) => fixed_rows(&local, positions),
        PlanarEquation::Midpoint => linear_equations::midpoint(&local),
        PlanarEquation::Distance {
            measure: DistanceMeasure::Horizontal | DistanceMeasure::Vertical,
            target,
            ..
        } => {
            let target = target_for(sketch, relation, target, targets, used, variables.len())?;
            let axis = if matches!(
                source.equation,
                PlanarEquation::Distance {
                    measure: DistanceMeasure::Horizontal,
                    ..
                }
            ) {
                0
            } else {
                1
            };
            linear_equations::axis_distance(&local, axis)
                .and_then(|value| Ok(vec![value.subtract_target(&target)?]))
        }
        _ => return Err(SketchIntervalError::Unsupported(relation)),
    }
    .map_err(|cause| SketchIntervalError::Arithmetic { relation, cause })?;
    Ok(values
        .into_iter()
        .enumerate()
        .map(|(equation, row)| SketchIntervalEquation {
            relation,
            equation,
            value: row.value.bounds(),
            jacobian: row.gradient.into_iter().map(Enclosure::bounds).collect(),
        })
        .collect())
}

fn fixed_rows(
    values: &[IntervalDual],
    positions: &[[f64; 2]],
) -> Result<Vec<IntervalDual>, String> {
    let count = values.first().map_or(0, |value| value.gradient.len());
    let targets = positions
        .iter()
        .flatten()
        .map(|&value| IntervalDual::constant(Enclosure::new([value, value])?, count))
        .collect::<Result<Vec<_>, String>>()?;
    linear_equations::fixed(values, &targets)
}
fn target_for(
    sketch: &Sketch2D,
    relation: SketchRelationId,
    prepared: f64,
    targets: &[SketchIntervalTarget],
    used: &mut [bool],
    count: usize,
) -> Result<IntervalDual, SketchIntervalError> {
    let SketchRelationId::Authored(id) = relation else {
        return Err(SketchIntervalError::Unsupported(relation));
    };
    let parameter = sketch
        .dimensional_constraints()
        .find(|(constraint, _)| constraint.id == id)
        .map(|(_, parameter)| parameter)
        .ok_or(SketchIntervalError::Unsupported(relation))?;
    let matches: Vec<_> = targets
        .iter()
        .enumerate()
        .filter(|(_, input)| input.relation == relation && input.parameter == parameter)
        .collect();
    let [(index, input)] = matches.as_slice() else {
        return Err(SketchIntervalError::Input(format!(
            "interval target {relation:?}, parameter {parameter}: expected exactly one matching bound"
        )));
    };
    let enclosure = Enclosure::new(input.bounds).map_err(SketchIntervalError::Input)?;
    if prepared < input.bounds[0] || prepared > input.bounds[1] {
        return Err(SketchIntervalError::Input(format!(
            "interval target {:?} for {relation:?}: expected prepared target {prepared:?} inside bounds",
            input.bounds
        )));
    }
    used[*index] = true;
    IntervalDual::constant(enclosure, count).map_err(SketchIntervalError::Input)
}
