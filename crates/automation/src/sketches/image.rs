// SPDX-License-Identifier: GPL-2.0-only
//! Route typed Sketch Image insertions through the document-owned history.
use super::*;
use oblikovati_model::{document_sketch::SketchImage, quantity::Dimension, sketch::SketchEntityId};

pub(super) fn add<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddSketchImageArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Add sketch image",
            |snapshot, ids| {
                let id = ids.allocate_sketch_entity().map_err(api_error)?;
                let image = build_image(snapshot, id, &request)?;
                snapshot
                    .sketches
                    .add_image(sketch, image)
                    .map_err(api_error)?;
                Ok(id)
            },
            |_, id| encode(&AddSketchImageResult { entity_id: id.0 }),
        )
        .map_err(api_error)
}

fn build_image(
    snapshot: &DocumentSnapshot,
    id: SketchEntityId,
    request: &AddSketchImageArgs,
) -> Result<SketchImage, Error> {
    let anchor = match request.anchor.as_deref() {
        Some(coords) if !coords.is_empty() => coordinates::working_pair(
            coords,
            snapshot.parameters.units().working_scale(),
            "sketch image anchor",
        )?,
        _ => [0.0, 0.0],
    };
    let width = centimetres(snapshot, &request.width)?;
    let height = centimetres(snapshot, &request.height)?;
    let rotation = if request.rotation.trim().is_empty() {
        0.0
    } else {
        snapshot
            .parameters
            .evaluate_scalar_expression(&request.rotation, Dimension::Angle)
            .map_err(api_error)?
            .value
    };
    let opacity = if request.opacity == 0.0 {
        1.0
    } else {
        request.opacity
    };
    let image = SketchImage {
        id,
        reference: request.ref_.clone(),
        anchor,
        width,
        height,
        rotation,
        opacity,
    };
    image.validate().map_err(api_error)?;
    Ok(image)
}

fn centimetres(snapshot: &DocumentSnapshot, input: &str) -> Result<f64, Error> {
    let length = snapshot
        .parameters
        .evaluate_scalar_expression(input, Dimension::Length)
        .map_err(api_error)?;
    Ok(length.value * snapshot.parameters.units().working_scale())
}
