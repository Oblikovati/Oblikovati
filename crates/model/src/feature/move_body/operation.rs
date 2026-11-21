// SPDX-License-Identifier: GPL-2.0-only
//! Validate and resolve each ordered body motion (#4097).
use super::*;

pub(super) fn validate(operation: MoveBodyOperation) -> Result<(), String> {
    match operation {
        MoveBodyOperation::Translation(point) => validate_lengths(&point),
        MoveBodyOperation::AlongRay {
            direction,
            distance,
        } => {
            validate_direction(direction)?;
            validate_lengths(&[distance])
        }
        MoveBodyOperation::Rotation {
            origin,
            direction,
            angle,
        } => {
            validate_direction(direction)?;
            validate_lengths(&origin)?;
            if angle.parameter_id() == Some(0) {
                return Err("move angle parameter 0: expected a nonzero id".into());
            }
            angle
                .validate_source()
                .map_err(|error| format!("move angle {angle:?}: expected finite radians: {error}"))
        }
    }
}
fn validate_lengths(lengths: &[DatumLength]) -> Result<(), String> {
    for value in lengths {
        if value.parameter() == Some(0) {
            return Err("move length parameter 0: expected a nonzero id".into());
        }
        value.validate_source().map_err(|error| {
            format!("move length {value:?}: expected finite coordinates: {error}")
        })?;
    }
    Ok(())
}
fn validate_direction(direction: [f64; 3]) -> Result<(), String> {
    if direction.iter().all(|value| value.is_finite()) && direction.iter().any(|value| *value != 0.)
    {
        return Ok(());
    }
    Err(format!(
        "move direction {direction:?}: expected a finite nonzero vector"
    ))
}
pub(super) fn parameters(operation: MoveBodyOperation) -> Vec<u64> {
    let mut result = Vec::new();
    match operation {
        MoveBodyOperation::Translation(point) => {
            result.extend(point.into_iter().filter_map(DatumLength::parameter))
        }
        MoveBodyOperation::AlongRay { distance, .. } => result.extend(distance.parameter()),
        MoveBodyOperation::Rotation { origin, angle, .. } => {
            result.extend(origin.into_iter().filter_map(DatumLength::parameter));
            result.extend(angle.parameter_id());
        }
    }
    result
}
pub(super) fn resolve(
    operation: MoveBodyOperation,
    parameters: &ParameterCollection<u64>,
) -> Result<ResolvedBodyMotion, String> {
    match operation {
        MoveBodyOperation::Translation(point) => Ok(ResolvedBodyMotion::Translation(
            resolve_point(point, parameters)?,
        )),
        MoveBodyOperation::AlongRay {
            direction,
            distance,
        } => {
            let vector = direction_unit(direction);
            let distance = length(distance, parameters)?;
            let translation = vector.map(|value| value * distance);
            if translation.iter().any(|value| !value.is_finite()) {
                return Err(format!(
                    "move direction {direction:?}, distance {distance}: expected finite translation"
                ));
            }
            Ok(ResolvedBodyMotion::Translation(translation))
        }
        MoveBodyOperation::Rotation {
            origin,
            direction,
            angle,
        } => Ok(ResolvedBodyMotion::Rotation {
            origin: resolve_point(origin, parameters)?,
            direction: direction_unit(direction),
            radians: angle.resolve(parameters).map_err(|error| {
                format!("move angle {angle:?}: expected finite radians: {error}")
            })?,
        }),
    }
}
fn length(value: DatumLength, parameters: &ParameterCollection<u64>) -> Result<f64, String> {
    value
        .resolve(parameters)
        .map_err(|error| format!("move length {value:?}: expected a finite length: {error}"))
}
fn resolve_point(
    point: [DatumLength; 3],
    parameters: &ParameterCollection<u64>,
) -> Result<[f64; 3], String> {
    Ok([
        length(point[0], parameters)?,
        length(point[1], parameters)?,
        length(point[2], parameters)?,
    ])
}
fn direction_unit(direction: [f64; 3]) -> [f64; 3] {
    let scale = direction.into_iter().map(f64::abs).fold(0., f64::max);
    let scaled = direction.map(|value| value / scale);
    let magnitude = scaled
        .into_iter()
        .map(|value| value * value)
        .sum::<f64>()
        .sqrt();
    scaled.map(|value| value / magnitude)
}
pub(super) fn same(a: MoveBodyOperation, b: MoveBodyOperation) -> bool {
    match (a, b) {
        (MoveBodyOperation::Translation(a), MoveBodyOperation::Translation(b)) => {
            same_lengths(a, b)
        }
        (
            MoveBodyOperation::AlongRay {
                direction: a,
                distance: x,
            },
            MoveBodyOperation::AlongRay {
                direction: b,
                distance: y,
            },
        ) => same_numbers(a, b) && x.same_recipe(y),
        (
            MoveBodyOperation::Rotation {
                origin: a,
                direction: x,
                angle: r,
            },
            MoveBodyOperation::Rotation {
                origin: b,
                direction: y,
                angle: s,
            },
        ) => same_lengths(a, b) && same_numbers(x, y) && r.same_recipe(s),
        _ => false,
    }
}
fn same_numbers(a: [f64; 3], b: [f64; 3]) -> bool {
    a.into_iter()
        .zip(b)
        .all(|(x, y)| x.to_bits() == y.to_bits())
}
fn same_lengths(a: [DatumLength; 3], b: [DatumLength; 3]) -> bool {
    a.into_iter().zip(b).all(|(x, y)| x.same_recipe(y))
}
