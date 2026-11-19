// SPDX-License-Identifier: GPL-2.0-only
//! Translate API scope requests to validated document-session candidates (#4107).
use oblikovati_api::{
    Error,
    wire::{SketchInferenceScopeArgs, SketchInferenceScopeView},
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchInferenceCandidates},
};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

fn addressed_sketch<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    index: i64,
) -> Result<(u64, DocumentSketchId), Error> {
    let document = workspace.active_document().ok_or_else(|| {
        Error("active document is absent: expected an active sketch document".into())
    })?;
    let snapshot = workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot();
    let sketch = crate::sketches::sketch_at(snapshot, index)?;
    Ok((document, sketch))
}
fn response(index: i64, candidates: &SketchInferenceCandidates) -> SketchInferenceScopeView {
    SketchInferenceScopeView {
        sketch_index: index,
        scope: candidates.mode().clone(),
        entities: Some(candidates.selected().iter().map(|id| id.0).collect()),
    }
}
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    args: SketchInferenceScopeArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = addressed_sketch(workspace, args.sketch_index)?;
    let candidates = workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?
        .sketch_inference_scope(sketch)
        .map_err(|error| Error(error.to_string()))?;
    crate::encode(&response(args.sketch_index, &candidates))
}
pub(super) fn prepare<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: SketchInferenceScopeView,
) -> Result<(u64, DocumentSketchId, SketchInferenceCandidates, Vec<u8>), Error> {
    let (document, sketch) = addressed_sketch(workspace, request.sketch_index)?;
    let entities: Vec<_> = request
        .entities
        .unwrap_or_default()
        .into_iter()
        .map(SketchEntityId)
        .collect();
    let candidates = SketchInferenceCandidates::new(request.scope, &entities)
        .map_err(|error| Error(error.to_string()))?;
    let geometry = workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .sketches
        .get(sketch)
        .map_err(|error| Error(error.to_string()))?
        .geometry();
    candidates
        .validate_geometry(geometry)
        .map_err(|error| Error(error.to_string()))?;
    let encoded = crate::encode(&response(request.sketch_index, &candidates))?;
    Ok((document, sketch, candidates, encoded))
}
