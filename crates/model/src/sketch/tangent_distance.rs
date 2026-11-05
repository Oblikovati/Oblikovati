// SPDX-License-Identifier: GPL-2.0-only
use super::{
    LineContactSide, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
};
use super::{tangent, tangent_residual::radial_projection};
/// Offset convention; near distance is signed when the line crosses the circle.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TangentDistanceOptions {
    pub far_side: bool,
    pub linear_diameter: bool,
}
/// Retained direction of the radius along the oriented line at a shared point.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum TangentLean {
    Positive,
    Negative,
}
/// Authored branch identity; lean exists exactly for a shared defining endpoint.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TangentDistanceContact {
    pub side: LineContactSide,
    pub lean: Option<TangentLean>,
}
impl Sketch2D {
    /// Retain the picked offset branch; example: `sketch.add_tangent_distance(id, line, arc, parameter, Default::default())?`.
    pub fn add_tangent_distance(
        &mut self,
        id: SketchConstraintId,
        line: SketchEntityId,
        curve: SketchEntityId,
        parameter: u64,
        options: TangentDistanceOptions,
    ) -> Result<(), SketchError> {
        let at_b = self.shared_line_contact(line, curve)?;
        let values = self.line_curve_values(line, curve)?;
        let contact =
            select_contact(&values, at_b).map_err(|cause| SketchError::ContactSelection {
                constraint: id,
                cause,
            })?;
        self.add_constraint(
            id,
            PlanarConstraint::TangentDistance {
                line,
                curve,
                parameter,
                options,
                contact,
            },
        )
    }
}
fn select_contact(
    values: &[f64],
    at_b: Option<bool>,
) -> Result<TangentDistanceContact, oblikovati_solve::residual::ResidualError> {
    let side = tangent::select_line_side(values)?;
    let variables = tangent::constants(values)?;
    let lean = at_b
        .map(|at_b| {
            radial_projection(&variables, at_b).map(|dot| {
                if dot.value() < 0.0 {
                    TangentLean::Negative
                } else {
                    TangentLean::Positive
                }
            })
        })
        .transpose()?;
    Ok(TangentDistanceContact { side, lean })
}
