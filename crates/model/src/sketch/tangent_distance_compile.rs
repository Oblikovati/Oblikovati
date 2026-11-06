// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraint, SketchEntityId, SketchError,
    TangentDistanceOptions, TangentLean,
};
use super::{
    compile::{self, DimensionDomain},
    residual::PlanarEquation,
};
use crate::{parameter::ParameterCollection, quantity::Dimension};
use std::f64::consts::PI;
pub(super) fn prepare_distance(
    sketch: &Sketch2D,
    constraint: &SketchConstraint,
    parameters: &ParameterCollection<u64>,
) -> Result<PlanarEquation, SketchError> {
    let PlanarConstraint::TangentDistance {
        line,
        curve,
        parameter,
        options,
        contact,
    } = constraint.relation
    else {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} relation {:?}: expected a tangent-distance dimension",
            constraint.id, constraint.relation
        )));
    };
    let target = linear_target(constraint, parameter, parameters, options)?;
    let at_b = sketch.shared_line_contact(line, curve)?;
    if let Some(at_b) = at_b
        && let Some(radius) = driving_radius(sketch, curve, parameters)?
    {
        let ratio = if options.far_side {
            target / radius - 1.0
        } else {
            1.0 + target / radius
        };
        let angle = shared_angle(
            constraint,
            ratio,
            contact.lean.ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch constraint {:?} shared contact {contact:?}: expected a retained lean",
                    constraint.id
                ))
            })?,
        )?;
        return Ok(PlanarEquation::PinnedTangentDistance {
            target: angle,
            at_b,
            side: contact.side,
        });
    }
    Ok(PlanarEquation::TangentDistance {
        target,
        far_side: options.far_side,
        contact,
        at_b,
    })
}
fn linear_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
    options: TangentDistanceOptions,
) -> Result<f64, SketchError> {
    let value = if options.far_side {
        compile::positive_length_target(constraint, parameter, parameters, "far offset")?
    } else {
        compile::dimensional_target(
            constraint,
            parameter,
            parameters,
            DimensionDomain {
                kind: Dimension::Length,
                minimum: f64::NEG_INFINITY,
                maximum: f64::INFINITY,
                label: "near offset",
                expected: "a healthy finite signed working-unit length",
            },
        )?
    };
    if !options.linear_diameter {
        return Ok(value);
    }
    compile::half_dimension(
        constraint,
        parameter,
        value,
        "a representable half-dimension",
    )
}

fn driving_radius(
    sketch: &Sketch2D,
    curve: SketchEntityId,
    parameters: &ParameterCollection<u64>,
) -> Result<Option<f64>, SketchError> {
    let mut radii = Vec::new();
    for constraint in &sketch.constraints {
        match constraint.relation {
            PlanarConstraint::Radius {
                curve: owner,
                parameter,
            } if owner == curve => radii.push(compile::positive_length_target(
                constraint, parameter, parameters, "radius",
            )?),
            PlanarConstraint::Diameter {
                curve: owner,
                parameter,
            } if owner == curve => {
                radii.push(compile::diameter_radius(constraint, parameter, parameters)?)
            }
            _ => {}
        }
    }
    radii.sort_by(f64::total_cmp);
    radii.dedup();
    Ok(if radii.len() == 1 {
        Some(radii[0])
    } else {
        None
    })
}
fn shared_angle(
    constraint: &SketchConstraint,
    ratio: f64,
    lean: TangentLean,
) -> Result<f64, SketchError> {
    if !(0.0..=1.0).contains(&ratio) {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} shared offset/radius ratio {ratio:?}: expected [0, 1] for a line through the defining contact",
            constraint.id
        )));
    }
    // Fixed radius permits a simple angular root without perturbing the contact (#3714).
    let angle = ratio.atan2(((1.0 - ratio) * (1.0 + ratio)).sqrt());
    Ok(if lean == TangentLean::Positive {
        angle
    } else {
        PI - angle
    })
}
