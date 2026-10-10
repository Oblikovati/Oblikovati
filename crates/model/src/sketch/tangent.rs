// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError, tangent_residual,
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

/// Retained side of the oriented line A→B, never reselected during iteration.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LineContactSide {
    Left,
    Right,
}
/// Retained external contact or explicit enclosing-circle identity by pair order.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CircleContact {
    External,
    FirstContainsSecond,
    SecondContainsFirst,
}
impl Sketch2D {
    /// Select and retain the current contact side; example: `sketch.add_line_tangent(id, line, circle)?`.
    pub fn add_line_tangent(
        &mut self,
        id: SketchConstraintId,
        line: SketchEntityId,
        circle: SketchEntityId,
    ) -> Result<(), SketchError> {
        let values = self.line_circle_values(line, circle)?;
        let side = select_line_side(&values).map_err(|cause| selection_error(id, cause))?;
        self.add_constraint(id, PlanarConstraint::LineTangent { line, circle, side })
    }
    /// Select the nearer contact branch once (external wins exact ties); example: `sketch.add_circle_tangent(id, [first, second])?`.
    pub fn add_circle_tangent(
        &mut self,
        id: SketchConstraintId,
        circles: [SketchEntityId; 2],
    ) -> Result<(), SketchError> {
        let values = self.circle_pair_values(circles)?;
        let contact = select_circle_contact(&values).map_err(|cause| selection_error(id, cause))?;
        self.add_constraint(id, PlanarConstraint::CircleTangent { circles, contact })
    }
    fn line_circle_values(
        &self,
        line: SketchEntityId,
        circle: SketchEntityId,
    ) -> Result<Vec<f64>, SketchError> {
        let endpoints = self.line(line)?.endpoints;
        let circle = self.circle(circle)?;
        let mut values = Vec::new();
        for point in [endpoints[0], endpoints[1], circle.center] {
            values.extend(self.point(point)?.position);
        }
        values.push(circle.radius);
        Ok(values)
    }
    fn circle_pair_values(&self, circles: [SketchEntityId; 2]) -> Result<Vec<f64>, SketchError> {
        let first = self.circle(circles[0])?;
        let second = self.circle(circles[1])?;
        let mut values = self.point(first.center)?.position.to_vec();
        values.extend(self.point(second.center)?.position);
        values.extend([first.radius, second.radius]);
        Ok(values)
    }
}
fn constants(values: &[f64]) -> Result<Vec<DualNumber>, ResidualError> {
    Ok(values
        .iter()
        .map(|value| DualNumber::constant(*value))
        .collect::<Result<_, _>>()?)
}
fn select_line_side(values: &[f64]) -> Result<LineContactSide, ResidualError> {
    let variables = constants(values)?;
    let signed = tangent_residual::signed_line_distance(&variables)?.value();
    if signed == 0.0 {
        return Err(ResidualError::Domain("line contact signed distance 0.0: expected a nonzero seed side or an explicit Left/Right constraint".into()));
    }
    Ok(if signed > 0.0 {
        LineContactSide::Left
    } else {
        LineContactSide::Right
    })
}
fn select_circle_contact(values: &[f64]) -> Result<CircleContact, ResidualError> {
    let variables = constants(values)?;
    let first = &variables[4];
    let second = &variables[5];
    let distance = super::residual::point(&variables, 0)?
        .minus(&super::residual::point(&variables, 2)?)?
        .length()?
        .value();
    let external = first.plus(second)?.value();
    let internal = first.minus(second)?.abs()?.value();
    let contact = if (distance - internal).abs() < (distance - external).abs() {
        if first.value() > second.value() {
            CircleContact::FirstContainsSecond
        } else {
            CircleContact::SecondContainsFirst
        }
    } else {
        CircleContact::External
    };
    tangent_residual::circle_tangent(&variables, contact)?;
    Ok(contact)
}
fn selection_error(id: SketchConstraintId, cause: ResidualError) -> SketchError {
    SketchError::ContactSelection {
        constraint: id,
        cause,
    }
}
