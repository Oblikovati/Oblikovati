// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::SketchDimensionEdit,
    quantity::Dimension,
    sketch::{Sketch2D, SketchConstraintId, SketchDimensionLimits},
};
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: DriveDimensionArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let (dimension, edit) = prepare(
        geometry,
        snapshot.parameters.units().working_scale(),
        &request,
    )?;
    files
        .commands(document)
        .map_err(api_error)?
        .edit_sketch_dimension_projected(sketch, dimension, &edit, encode_edited)
        .map_err(api_error)
}
fn prepare(
    geometry: &Sketch2D,
    scale: f64,
    request: &DriveDimensionArgs,
) -> Result<(SketchConstraintId, SketchDimensionEdit), Error> {
    let dimension = super::dimensions::selected_constraint(geometry, request.dimension_index)?;
    let (_, category) = geometry.dimension_source(dimension).map_err(api_error)?;
    let drive_limits = if request.set_limits {
        Some(convert_limits(request, category, scale)?)
    } else {
        None
    };
    let edit = SketchDimensionEdit {
        target_expression: (!request.expression.is_empty()).then(|| request.expression.clone()),
        reference: request.set_driven.then_some(request.driven),
        drive_limits,
    };
    Ok((dimension, edit))
}
fn convert_limits(
    request: &DriveDimensionArgs,
    category: Dimension,
    scale: f64,
) -> Result<SketchDimensionLimits, Error> {
    let (factor, units) = if category == Dimension::Length {
        (scale, "centimetres")
    } else {
        (1.0, "radians")
    };
    let [lower, upper] = super::coordinates::finite_metric_pair(
        [request.min, request.max],
        factor,
        "dimension drive limits",
        units,
    )?;
    let limits = SketchDimensionLimits { lower, upper };
    limits.validate().map_err(api_error)?;
    Ok(limits)
}

fn encode_edited(
    reply: Result<(), oblikovati_model::sketch::SketchError>,
) -> Result<Vec<u8>, Error> {
    reply.map_err(api_error)?;
    encode(&OKResult { ok: true })
}

#[cfg(test)]
#[path = "drive_dimension/tests.rs"]
mod tests;
