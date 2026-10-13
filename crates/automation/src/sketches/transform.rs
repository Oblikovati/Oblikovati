// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    sketch::{SketchEntityId, SketchSelectionTransform},
};
use oblikovati_session::SketchTransformProjectionInput;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: TransformSketchArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let parameters = &files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot()
        .parameters;
    let transform = prepared_transform(parameters, &request)?;
    let entities = selected_ids(request.entities)?;
    files
        .commands(document)
        .map_err(api_error)?
        .transform_sketch_entities_projected(sketch, &entities, transform, transform_reply)
        .map_err(api_error)
}

fn transform_reply(result: SketchTransformProjectionInput) -> Result<Vec<u8>, Error> {
    result.map_err(api_error)?;
    encode(&TransformSketchResult::default())
}

fn prepared_transform(
    parameters: &ParameterCollection<u64>,
    request: &TransformSketchArgs,
) -> Result<SketchSelectionTransform, Error> {
    let scale = parameters.units().working_scale();
    match request.op.as_str() {
        "move" => Ok(SketchSelectionTransform::Move(movement_vector(
            request, scale,
        )?)),
        "rotate" => rotation_transform(parameters, request),
        _ => Err(Error(format!(
            "sketch.transform operation {:?}: expected move or rotate; other transform operations are not ported",
            request.op
        ))),
    }
}
fn rotation_angle(parameters: &ParameterCollection<u64>, source: &str) -> Result<f64, Error> {
    if let Ok(quantity) = parameters.evaluate_expression(source)
        && quantity.dimension == Dimension::Angle
    {
        return Ok(quantity.value);
    }
    let quantity = parameters
        .units()
        .parse(source, Dimension::Angle)
        .map_err(|cause| Error(format!("sketch.transform angle {source:?}: {cause}")))?;
    checked_rotation_angle(source, quantity)
}
fn checked_rotation_angle(source: &str, quantity: Quantity) -> Result<f64, Error> {
    if quantity.dimension != Dimension::Angle || !quantity.value.is_finite() {
        return Err(Error(format!(
            "sketch.transform angle {source:?} resolved to {quantity:?}: expected a finite angle"
        )));
    }
    Ok(quantity.value)
}
fn rotation_transform(
    parameters: &ParameterCollection<u64>,
    request: &TransformSketchArgs,
) -> Result<SketchSelectionTransform, Error> {
    let center = super::coordinates::working_pair(
        request.center.as_deref().unwrap_or_default(),
        parameters.units().working_scale(),
        "sketch.transform center",
    )?;
    Ok(SketchSelectionTransform::Rotate {
        center,
        radians: rotation_angle(parameters, &request.angle)?,
    })
}

fn selected_ids(entities: Option<Vec<u64>>) -> Result<Vec<SketchEntityId>, Error> {
    let entities = entities.unwrap_or_default();
    if entities.is_empty() {
        return Err(Error(format!(
            "sketch.transform selection {entities:?}: expected at least one owned entity"
        )));
    }
    Ok(entities.into_iter().map(SketchEntityId).collect())
}

fn movement_vector(request: &TransformSketchArgs, scale: f64) -> Result<[f64; 2], Error> {
    super::coordinates::working_pair(
        request.vector.as_deref().unwrap_or_default(),
        scale,
        "sketch.transform vector",
    )
}
