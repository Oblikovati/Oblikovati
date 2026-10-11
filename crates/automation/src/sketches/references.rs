// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::{DocumentSketchReferenceIndex, SketchReferenceTarget};
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = owner::active_sketch_host(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let names = DocumentSketchReferenceIndex::new(&snapshot.sketches, guid).map_err(api_error)?;
    dispatch(snapshot, &names, method, request)
}
fn dispatch(
    snapshot: &DocumentSnapshot,
    names: &DocumentSketchReferenceIndex,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_SKETCH_REFERENCE_KEY => {
            reference_key(snapshot, names, crate::decode(method, request)?)
        }
        METHOD_SKETCH_RESOLVE_REFERENCE => {
            resolve_reference(snapshot, names, crate::decode(method, request)?)
        }
        _ => Err(Error(format!(
            "sketch reference query {method:?}: expected {METHOD_SKETCH_REFERENCE_KEY:?} or {METHOD_SKETCH_RESOLVE_REFERENCE:?}"
        ))),
    }
}
fn reference_key(
    snapshot: &DocumentSnapshot,
    names: &DocumentSketchReferenceIndex,
    request: SketchArgs,
) -> Result<Vec<u8>, Error> {
    let sketch = sketch_at(snapshot, request.sketch_index)?;
    encode(&SketchReferenceKeyResult {
        reference_key: names.sketch_key(sketch).map_err(api_error)?,
    })
}
fn resolve_reference(
    snapshot: &DocumentSnapshot,
    names: &DocumentSketchReferenceIndex,
    request: ResolveSketchReferenceArgs,
) -> Result<Vec<u8>, Error> {
    let Some(target) = names.resolve(&request.reference_key) else {
        return encode(&ResolveSketchReferenceResult::default());
    };
    encode(&resolved_target(snapshot, target)?)
}
fn resolved_target(
    snapshot: &DocumentSnapshot,
    target: SketchReferenceTarget,
) -> Result<ResolveSketchReferenceResult, Error> {
    let position = current_sketch_position(snapshot, target)?;
    Ok(ResolveSketchReferenceResult {
        found: true,
        kind: if target.entity.is_some() {
            "sketchEntity"
        } else {
            "sketch"
        }
        .into(),
        sketch_index: position,
        entity_id: target.entity.map_or(0, |entity| entity.0),
    })
}

fn current_sketch_position(
    snapshot: &DocumentSnapshot,
    target: SketchReferenceTarget,
) -> Result<i64, Error> {
    let position = snapshot
        .sketches
        .sketches()
        .iter()
        .position(|placed| placed.id() == target.sketch)
        .ok_or_else(|| {
            Error(format!(
                "sketch reference target {target:?}: expected a current owned sketch"
            ))
        })?;
    wire_count(position)
}
