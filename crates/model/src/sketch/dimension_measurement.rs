// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchConstraint, SketchConstraintId, SketchError};
use crate::quantity::Quantity;
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
impl Sketch2D {
    /// Measure an owned dimension from current stored geometry, independently of its parameter target.
    /// Example: `let measured = sketch.measure_dimension(constraint.id)?;` for an immutable inspection.
    pub fn measure_dimension(&self, id: SketchConstraintId) -> Result<Quantity, SketchError> {
        let constraint = self.dimensional_constraints().find(|(constraint, _)| constraint.id == id)
            .map(|(constraint, _)| constraint).ok_or_else(|| SketchError::Definition(format!(
                "sketch constraint {id:?}: expected an owned dimensional relation for measurement"
            )))?;
        let variables = self.dimension_measurement_variables(constraint)?;
        let value = measurement(self, &constraint.relation, &variables)
            .map_err(|cause| measurement_error(id, cause))?;
        let dimension = super::dependencies::dimension_category(&constraint.relation);
        Ok(Quantity::new(value.value(), dimension))
    }
    fn dimension_measurement_variables(
        &self,
        constraint: &SketchConstraint,
    ) -> Result<Vec<DualNumber>, SketchError> {
        let (projection, columns) =
            super::circular_compile::relation_projection(self, &constraint.relation, None)?;
        let values = self.variable_values();
        let constants = columns
            .into_iter()
            .map(|column| DualNumber::constant(values[column]))
            .collect::<Result<Vec<_>, _>>()
            .map_err(ResidualError::from)
            .map_err(|cause| measurement_error(constraint.id, cause))?;
        let Some(projection) = projection else {
            return Ok(constants);
        };
        projection
            .project(&constants)
            .map_err(|cause| measurement_error(constraint.id, cause))
    }
}
fn measurement(
    sketch: &Sketch2D,
    relation: &PlanarConstraint,
    variables: &[DualNumber],
) -> Result<DualNumber, ResidualError> {
    use PlanarConstraint::*;
    match relation {
        ArcLength { arc, .. } => super::arc_length::length_measure(
            variables,
            sketch
                .arc(*arc)
                .map_err(|cause| ResidualError::Domain(cause.to_string()))?
                .counter_clockwise,
        ),
        Angle { .. } | ThreePointAngle { .. } => Ok(super::angle::angle_measure(variables)?.abs()?),
        TangentDistance { options, .. } => tangent_measurement(variables, *options),
        _ => linear_measurement(relation, variables),
    }
}
fn tangent_measurement(
    variables: &[DualNumber],
    options: super::TangentDistanceOptions,
) -> Result<DualNumber, ResidualError> {
    let distance = super::tangent_residual::signed_line_distance(variables)?.abs()?;
    let offset = super::tangent_distance_residual::circular_offset(
        &distance,
        &variables[6],
        options.far_side,
    )?;
    diameter_value(offset, options.linear_diameter)
}
fn diameter_value(measured: DualNumber, diameter: bool) -> Result<DualNumber, ResidualError> {
    if diameter {
        return Ok(measured.scaled(2.0)?);
    }
    Ok(measured)
}
fn measurement_error(id: SketchConstraintId, cause: ResidualError) -> SketchError {
    SketchError::Definition(format!(
        "dimension {id:?}: expected a finite defined geometric measurement: {cause}"
    ))
}

fn linear_measurement(
    relation: &PlanarConstraint,
    variables: &[DualNumber],
) -> Result<DualNumber, ResidualError> {
    use PlanarConstraint::*;
    match relation {
        Distance { measure, .. } => super::residual::distance(variables, *measure),
        Radius { .. } => Ok(variables[0].clone()),
        Diameter { .. } => Ok(variables[0].scaled(2.0)?),
        Offset {
            linear_diameter, ..
        } => diameter_value(
            super::line_residual::signed_offset(variables, 2, 0)?.abs()?,
            *linear_diameter,
        ),
        _ => Err(ResidualError::Domain(format!(
            "measurement relation {relation:?}: expected a dimensional relation"
        ))),
    }
}
