// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::{
    document_sketch::{SketchOrientationDefinition, SketchPlacement},
    history::DocumentSketchCreation,
    sketch::{SketchFrameAxis, SketchPlane},
};
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: CreateSketchArgs,
    project_origin: bool,
) -> Result<Vec<u8>, Error> {
    let document = super::owner::active_sketch_host(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let project_origin =
        project_origin && files.document_type(document).map_err(api_error)? == DOCUMENT_PART;
    let (definition, label) = creation_inputs(snapshot, request, project_origin)?;
    let index = wire_count(snapshot.sketches.sketches().len())?;
    files
        .commands(document)
        .map_err(api_error)?
        .create_sketch_projected(&definition, |result| creation_reply(result, index, label))
        .map_err(api_error)
}
fn creation_inputs(
    snapshot: &DocumentSnapshot,
    request: CreateSketchArgs,
    project_origin: bool,
) -> Result<(DocumentSketchCreation, String), Error> {
    let (placement, label) = match request.work_plane_index {
        Some(index) => crate::datums::sketch_plane_at_index(snapshot, *index)?,
        None => origin_plane(&request.plane)?,
    };
    let orientation = request
        .orientation
        .as_deref()
        .map(|input| orientation_input(snapshot, input))
        .transpose()?;
    Ok((
        DocumentSketchCreation {
            name: "Sketch".into(),
            placement,
            orientation,
            project_origin,
        },
        label,
    ))
}
fn origin_plane(label: &str) -> Result<(SketchPlacement, String), Error> {
    let name = label.trim().to_ascii_uppercase();
    let (frame, normalized) = match name.as_str() {
        "" | "XY" => (SketchPlane::xy(), "XY"),
        "XZ" => (SketchPlane::xz(), "XZ"),
        "YZ" => (SketchPlane::yz(), "YZ"),
        _ => {
            return Err(Error(format!(
                "sketch.create origin plane {label:?}: expected XY, XZ, YZ, or empty for XY"
            )));
        }
    };
    Ok((SketchPlacement::Fixed(frame), normalized.into()))
}
fn orientation_input(
    snapshot: &DocumentSnapshot,
    input: &SketchOrientation,
) -> Result<SketchOrientationDefinition, Error> {
    let origin = input.origin.as_deref().filter(|values| !values.is_empty()).map(|values| {
        let coordinates = <[f64; 3]>::try_from(values).map_err(|_| Error(format!("sketch.create orientation origin {values:?}: expected three finite coordinates or empty")))?;
        Ok::<_, Error>(coordinates)
    }).transpose()?;
    Ok(SketchOrientationDefinition {
        reference: crate::datums::axis_reference(snapshot, &input.axis).map_err(|cause| Error(format!("sketch.create orientation axis {:?}: expected an available origin or datum axis: {cause}", input.axis)))?,
        sketch_axis: if input.axis_is_x {
            SketchFrameAxis::X
        } else {
            SketchFrameAxis::Y
        },
        reverse: input.reverse,
        origin,
    })
}

fn creation_reply(
    result: oblikovati_session::SketchCreationProjectionInput,
    index: i64,
    plane: String,
) -> Result<Vec<u8>, Error> {
    result.map_err(api_error)?;
    encode(&CreateSketchResult {
        sketch_index: index,
        plane,
    })
}
