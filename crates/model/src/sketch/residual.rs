// SPDX-License-Identifier: GPL-2.0-only
use super::{DistanceMeasure, LineRelation, line_residual};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::{ResidualError, ResidualSource},
};

pub(super) enum PlanarEquation {
    Coincident,
    Coordinate(usize),
    Fix([f64; 2]),
    Midpoint,
    PointOnLine,
    Lines(LineRelation),
    Distance {
        measure: DistanceMeasure,
        target: f64,
    },
}
pub(super) struct PreparedConstraint {
    pub equation: PlanarEquation,
    pub variables: Vec<usize>,
}
impl PreparedConstraint {
    pub fn equations(&self) -> usize {
        match self.equation {
            PlanarEquation::Coincident
            | PlanarEquation::Fix(_)
            | PlanarEquation::Midpoint
            | PlanarEquation::Lines(LineRelation::Collinear) => 2,
            _ => 1,
        }
    }
    fn variable_count(&self) -> usize {
        match self.equation {
            PlanarEquation::Fix(_) => 2,
            PlanarEquation::Midpoint | PlanarEquation::PointOnLine => 6,
            PlanarEquation::Lines(_) => 8,
            _ => 4,
        }
    }
}
impl ResidualSource for PreparedConstraint {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        if variables.len() != self.variable_count() {
            return Err(ResidualError::Domain(format!(
                "planar residual received {} variables: expected {}",
                variables.len(),
                self.variable_count()
            )));
        }
        match self.equation {
            PlanarEquation::Coincident => Ok(point(variables, 0)?
                .minus(&point(variables, 2)?)?
                .components()
                .to_vec()),
            PlanarEquation::Coordinate(axis) => {
                Ok(vec![variables[axis].minus(&variables[axis + 2])?])
            }
            PlanarEquation::Fix(position) => Ok(vec![
                variables[0].shifted(-position[0])?,
                variables[1].shifted(-position[1])?,
            ]),
            PlanarEquation::Midpoint => midpoint(variables),
            PlanarEquation::PointOnLine => line_residual::point_on_line(variables),
            PlanarEquation::Lines(relation) => line_residual::line_relation(variables, relation),
            PlanarEquation::Distance { measure, target } => {
                Ok(vec![distance(variables, measure)?.shifted(-target)?])
            }
        }
    }
}
pub(super) fn point(
    variables: &[DualNumber],
    offset: usize,
) -> Result<DualVector<2>, ResidualError> {
    Ok(DualVector::new([
        variables[offset].clone(),
        variables[offset + 1].clone(),
    ])?)
}
fn midpoint(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    // Scale before summing so a representable midpoint does not overflow (#3704).
    let middle = point(variables, 2)?
        .scaled(0.5)?
        .plus(&point(variables, 4)?.scaled(0.5)?)?;
    Ok(point(variables, 0)?.minus(&middle)?.components().to_vec())
}
fn distance(
    variables: &[DualNumber],
    measure: DistanceMeasure,
) -> Result<DualNumber, ResidualError> {
    match measure {
        DistanceMeasure::Aligned => Ok(point(variables, 0)?
            .minus(&point(variables, 2)?)?
            .length()?),
        DistanceMeasure::Horizontal => Ok(variables[0].minus(&variables[2])?.abs()?),
        DistanceMeasure::Vertical => Ok(variables[1].minus(&variables[3])?.abs()?),
    }
}

#[cfg(test)]
#[path = "../../tests/sketch/residuals.rs"]
mod tests;
