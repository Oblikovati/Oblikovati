// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError, tangent_residual,
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

/// Retained side of the oriented line A→B, never reselected during iteration.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum LineContactSide {
    Left,
    Right,
}
/// Retained external contact or explicit enclosing circular-curve identity by pair order.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum CircleContact {
    External,
    FirstContainsSecond,
    SecondContainsFirst,
}
impl Sketch2D {
    /// Select and retain the current contact side; example: `sketch.add_line_tangent(id, line, curve)?`.
    pub fn add_line_tangent(
        &mut self,
        id: SketchConstraintId,
        line: SketchEntityId,
        curve: SketchEntityId,
    ) -> Result<(), SketchError> {
        self.shared_line_contact(line, curve)?;
        let values = self.line_curve_values(line, curve)?;
        let side = select_line_side(&values).map_err(|cause| selection_error(id, cause))?;
        self.add_constraint(id, PlanarConstraint::LineTangent { line, curve, side })
    }
    /// Retain the seed contact branch; shared points use radial orientation, otherwise the nearer target (external wins ties); example: `sketch.add_circular_tangent(id, [first, second])?`.
    pub fn add_circular_tangent(
        &mut self,
        id: SketchConstraintId,
        curves: [SketchEntityId; 2],
    ) -> Result<(), SketchError> {
        let touch = self.shared_circular_contact(curves)?;
        let values = self.circular_pair_values(curves)?;
        let contact = if let Some(touch) = touch {
            let point = self.point(touch)?.position;
            select_pinned_contact(&values, point)
        } else {
            select_circle_contact(&values)
        }
        .map_err(|cause| selection_error(id, cause))?;
        self.add_constraint(id, PlanarConstraint::CircularTangent { curves, contact })
    }
    pub(super) fn line_curve_values(
        &self,
        line: SketchEntityId,
        curve: SketchEntityId,
    ) -> Result<Vec<f64>, SketchError> {
        let endpoints = self.line(line)?.endpoints;
        let (center, radius) = self.frame_values(self.circular_frame(curve)?)?;
        let mut values = Vec::new();
        for point in endpoints {
            values.extend(self.point(point)?.position);
        }
        values.extend(center);
        values.push(radius);
        Ok(values)
    }
    fn circular_pair_values(&self, curves: [SketchEntityId; 2]) -> Result<Vec<f64>, SketchError> {
        let first = self.frame_values(self.circular_frame(curves[0])?)?;
        let second = self.frame_values(self.circular_frame(curves[1])?)?;
        let mut values = first.0.to_vec();
        values.extend(second.0);
        values.extend([first.1, second.1]);
        Ok(values)
    }
}
pub(super) fn constants(values: &[f64]) -> Result<Vec<DualNumber>, ResidualError> {
    Ok(values
        .iter()
        .map(|value| DualNumber::constant(*value))
        .collect::<Result<_, _>>()?)
}
pub(super) fn select_line_side(values: &[f64]) -> Result<LineContactSide, ResidualError> {
    let variables = constants(values)?;
    let signed = tangent_residual::signed_line_distance(&variables)?.value();
    select_signed_side(signed)
}
pub(super) fn select_signed_side(signed: f64) -> Result<LineContactSide, ResidualError> {
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

fn select_pinned_contact(values: &[f64], touch: [f64; 2]) -> Result<CircleContact, ResidualError> {
    let mut values = values.to_vec();
    values.extend(touch);
    let variables = constants(&values)?;
    let dot = tangent_residual::radial_contact_dot(&variables)?.value();
    let contact = if dot <= 0.0 {
        CircleContact::External
    } else if values[4] > values[5] {
        CircleContact::FirstContainsSecond
    } else {
        CircleContact::SecondContainsFirst
    };
    tangent_residual::pinned_circular_tangent(&variables, contact)?;
    Ok(contact)
}
