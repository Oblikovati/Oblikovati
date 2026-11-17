// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchEntityId;
pub(super) fn corner_tangent_source(
    sketch: &Sketch2D,
    constraint: &SketchConstraint,
) -> Result<Option<PreparedConstraint>, SketchError> {
    let Some(owner) = sketch.owned_relations.get(&constraint.id) else {
        return Ok(None);
    };
    let PlanarConstraint::LineTangent { line, curve, side } = constraint.relation else {
        return Ok(None);
    };
    if curve != *owner {
        return Err(SketchError::Definition(format!(
            "owned tangent {constraint:?}: expected its curve to own the relation"
        )));
    }
    let corner = sketch
        .constraints
        .iter()
        .find_map(|row| {
            if sketch.owned_relations.get(&row.id) != Some(owner) {
                return None;
            }
            match row.relation {
                PlanarConstraint::PointOnLine {
                    point,
                    line: source,
                } if source == line => Some(point),
                _ => None,
            }
        })
        .ok_or_else(|| {
            SketchError::Definition(format!(
                "owned tangent {constraint:?}: expected a corresponding virtual corner"
            ))
        })?;
    let line_points = sketch.line(line)?.endpoints;
    let first = corner_trimmed_first(sketch, line, *owner)?;
    let (trim, far) = if first {
        (line_points[0], line_points[1])
    } else {
        (line_points[1], line_points[0])
    };
    let center = sketch.arc(curve)?.center;
    let variables = sketch
        .point_variables(&[corner, far, trim, center])
        .into_iter()
        .map(|variable| sketch.variable_column(variable))
        .collect::<Result<_, _>>()?;
    Ok(Some(PreparedConstraint {
        equation: PlanarEquation::CornerTangent {
            side,
            trimmed_first: first,
        },
        projection: None,
        variables,
    }))
}
pub(super) fn corner_trimmed_first(
    sketch: &Sketch2D,
    line: SketchEntityId,
    owner: SketchEntityId,
) -> Result<bool, SketchError> {
    let endpoints = sketch.line(line)?.endpoints;
    let blend = if let Ok(arc) = sketch.arc(owner) {
        [arc.start, arc.end]
    } else {
        sketch.line(owner)?.endpoints
    };
    match [blend.contains(&endpoints[0]), blend.contains(&endpoints[1])] {
        [true, false] => Ok(true),
        [false, true] => Ok(false),
        _ => Err(SketchError::Definition(format!(
            "corner line {line:?}, owner {owner:?}: expected exactly one shared blend endpoint"
        ))),
    }
}
