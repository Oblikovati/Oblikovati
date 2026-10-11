// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEntityId;
use oblikovati_session::SketchTranslationProjectionInput;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: TransformSketchArgs,
) -> Result<Vec<u8>, Error> {
    if request.op != "move" {
        return Err(Error(format!(
            "sketch.transform operation {:?}: expected move; other transform operations are not ported",
            request.op
        )));
    }
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let scale = working_scale(files, document)?;
    let vector = working_vector(request.vector.as_deref().unwrap_or_default(), scale)?;
    let entities = selected_ids(request.entities)?;
    files
        .commands(document)
        .map_err(api_error)?
        .translate_sketch_entities_projected(sketch, &entities, vector, movement_reply)
        .map_err(api_error)
}
fn working_vector(vector: &[f64], scale: f64) -> Result<[f64; 2], Error> {
    let pair = <[f64; 2]>::try_from(vector).map_err(|_| {
        Error(format!(
            "sketch.transform vector {vector:?}: expected two finite centimetre components"
        ))
    })?;
    let converted = pair.map(|value| value / scale);
    if pair
        .iter()
        .chain(converted.iter())
        .any(|value| !value.is_finite())
    {
        return Err(Error(format!(
            "sketch.transform vector {vector:?} scale {scale:?}: expected finite centimetres representable in working units"
        )));
    }
    Ok(converted)
}
fn movement_reply(result: SketchTranslationProjectionInput) -> Result<Vec<u8>, Error> {
    result.map_err(api_error)?;
    encode(&TransformSketchResult::default())
}

fn working_scale<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<f64, Error> {
    Ok(files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot()
        .parameters
        .units()
        .working_scale())
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
