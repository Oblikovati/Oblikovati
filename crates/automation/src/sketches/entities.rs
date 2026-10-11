// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    GeometryMoveableStatus, MOVEABLE_BY_DIMENSION_CHANGE, MOVEABLE_FIXED, MOVEABLE_FREE,
};
use oblikovati_model::{
    document_sketch::DocumentSketchReferenceIndex, sketch::SketchEntityInspection,
};
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let shapes = session.inspect_sketch_entities(sketch).map_err(api_error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let names =
        DocumentSketchReferenceIndex::new(&session.snapshot().sketches, guid).map_err(api_error)?;
    let scale = session.snapshot().parameters.units().working_scale();
    let entities = entity_rows(&names, sketch, shapes, scale)?;
    encode(&EnumerateEntitiesResult {
        entities: Some(entities),
    })
}
fn entity_rows(
    names: &DocumentSketchReferenceIndex,
    sketch: DocumentSketchId,
    shapes: Vec<SketchEntityInspection>,
    scale: f64,
) -> Result<Vec<SketchEntityInfo>, Error> {
    shapes
        .into_iter()
        .enumerate()
        .map(|(index, shape)| {
            let key = names.entity_key(sketch, shape.id).map_err(api_error)?;
            wire_entity(index, shape, key, scale)
        })
        .collect()
}
fn wire_entity(
    index: usize,
    shape: SketchEntityInspection,
    key: String,
    scale: f64,
) -> Result<SketchEntityInfo, Error> {
    let points = wire_points(&shape.points, scale)?;
    Ok(SketchEntityInfo {
        index: wire_count(index)?,
        id: shape.id.0,
        kind: shape.kind.0.into_owned(),
        points: Some(points),
        radius: centimetres(shape.radius, scale)?,
        reference_key: key,
        moveable_status: movement_name(shape.movement)?.into(),
        ..Default::default()
    })
}
fn wire_points(points: &[[f64; 2]], scale: f64) -> Result<Vec<Option<Vec<f64>>>, Error> {
    points
        .iter()
        .map(|point| {
            let coordinates = point
                .iter()
                .map(|value| centimetres(*value, scale))
                .collect::<Result<Vec<_>, Error>>()?;
            Ok(Some(coordinates))
        })
        .collect()
}
fn centimetres(value: f64, scale: f64) -> Result<f64, Error> {
    let converted = value * scale;
    if !converted.is_finite() {
        return Err(Error(format!(
            "sketch length {value:?} at scale {scale:?}: expected finite centimetres"
        )));
    }
    Ok(converted)
}
fn movement_name(status: GeometryMoveableStatus) -> Result<&'static str, Error> {
    match status {
        MOVEABLE_FREE => Ok("freeToMove"),
        MOVEABLE_FIXED => Ok("fixed"),
        MOVEABLE_BY_DIMENSION_CHANGE => Ok("byDimensionChange"),
        _ => Err(Error(format!(
            "sketch movement {status:?}: expected an evaluated primitive status"
        ))),
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn unclassified_movement_and_nonrepresentable_centimetres_are_refused() {
        assert!(
            movement_name(GeometryMoveableStatus(53508))
                .unwrap_err()
                .0
                .contains("53508")
        );
        let cause = centimetres(f64::MAX, 2.0).unwrap_err();
        assert!(cause.0.contains("scale 2.0"));
        assert!(cause.0.contains("finite centimetres"));
        assert_eq!(
            centimetres(-0.0, 2.0).unwrap().to_bits(),
            (-0.0_f64).to_bits()
        );
    }
}
