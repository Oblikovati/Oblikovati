// SPDX-License-Identifier: GPL-2.0-only
//! Convert the canonical ordered motion arguments to parameter-driven recipes (#4097).
use super::*;
use oblikovati_api::wire::featureargs::MoveBodyOp;
use oblikovati_model::{
    expression::Expression,
    feature::{
        move_body::MoveBodyOperation,
        work_geometry::{DatumAngle, DatumLength},
    },
    quantity::Dimension,
};

pub(super) fn operations(
    snapshot: &DocumentSnapshot,
    input: &MoveBody,
) -> Result<Vec<MoveBodyOperation>, Error> {
    if let Some(ops) = input.operations.as_ref().filter(|ops| !ops.is_empty()) {
        return ops
            .iter()
            .enumerate()
            .map(|(index, op)| {
                operation(snapshot, op)
                    .map_err(|error| Error(format!("moveBody operation {index}: {error}")))
            })
            .collect();
    }
    Ok(vec![MoveBodyOperation::Translation(
        vector(&input.translation, "translation")?.map(DatumLength::Constant),
    )])
}
fn operation(snapshot: &DocumentSnapshot, input: &MoveBodyOp) -> Result<MoveBodyOperation, Error> {
    match input.type_.as_str() {
        "freeDrag" => Ok(MoveBodyOperation::Translation([
            optional_length(snapshot, &input.x, "freeDrag x")?,
            optional_length(snapshot, &input.y, "freeDrag y")?,
            optional_length(snapshot, &input.z, "freeDrag z")?,
        ])),
        "alongRay" => Ok(MoveBodyOperation::AlongRay {
            direction: vector(&input.dir, "alongRay dir")?,
            distance: crate::extrusion::finite_length(snapshot, &input.dist, "alongRay dist")?,
        }),
        "rotateAboutLine" => Ok(MoveBodyOperation::Rotation {
            origin: vector(&input.point, "rotateAboutLine point")?.map(DatumLength::Constant),
            direction: vector(&input.dir, "rotateAboutLine dir")?,
            angle: angle(snapshot, &input.angle, "moveBody angle")?,
        }),
        other => Err(Error(format!(
            "moveBody operation type {other:?}: expected freeDrag, alongRay or rotateAboutLine"
        ))),
    }
}
fn vector(values: &Option<Vec<f64>>, label: &str) -> Result<[f64; 3], Error> {
    let result: [f64; 3] = values
        .as_deref()
        .unwrap_or_default()
        .try_into()
        .map_err(|_| {
            Error(format!(
                "moveBody {label} {values:?}: expected three finite values"
            ))
        })?;
    if result.iter().any(|value| !value.is_finite()) {
        return Err(Error(format!(
            "moveBody {label} {result:?}: expected three finite values"
        )));
    }
    Ok(result)
}
fn optional_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    if source.trim().is_empty() {
        return Ok(DatumLength::Constant(0.));
    }
    crate::extrusion::finite_length(snapshot, source, label)
}
pub(crate) fn angle(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumAngle, Error> {
    let parameters = &snapshot.parameters;
    let text = source.trim();
    if let Some(id) = parameters.find(text) {
        let result = DatumAngle::parameter(id);
        result.resolve(parameters).map_err(api_error)?;
        return Ok(result);
    }
    let expression = Expression::parse(text, parameters.units()).map_err(api_error)?;
    if !expression.references().is_empty() {
        return Err(Error(format!(
            "{label} {source:?}: expected a parameter name or a reference-free angle expression"
        )));
    }
    let mut quantity = expression.evaluate().map_err(api_error)?;
    if quantity.dimension == Dimension::Unitless {
        quantity = parameters
            .units()
            .from_preferred(quantity.value, Dimension::Angle);
    }
    if quantity.dimension != Dimension::Angle || !quantity.value.is_finite() {
        return Err(Error(format!(
            "{label} {source:?}: expected a finite angle"
        )));
    }
    DatumAngle::constant(quantity.value).map_err(api_error)
}
