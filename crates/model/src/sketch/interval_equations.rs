// SPDX-License-Identifier: GPL-2.0-only
//! Conditional equation enclosures; these results do not certify a solve.
mod dual;
mod enclosure;
use self::{dual::IntervalDual, enclosure::Enclosure};
use super::{
    Sketch2D, SketchError, SketchRelationId, compile, linear_equations,
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
                "interval relation {relation:?}: expected a supported coordinate equation"
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
        for (source, relation) in prepared.sources.iter().zip(prepared.origins) {
            rows.extend(source_rows(source, relation, &variables)?);
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
    source: &PreparedConstraint,
    relation: SketchRelationId,
    variables: &[IntervalDual],
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
