// SPDX-License-Identifier: GPL-2.0-only
//! Explode a placed instance through the typed sketch API.
use super::*;
use oblikovati_model::sketch::SketchEntityId;

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: ExplodeSketchBlockArgs,
) -> Result<Vec<u8>, Error> {
    let document = super::block_definitions::active_part(files)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Explode sketch block",
            |snapshot, reservations| {
                let sketch = sketch_at(snapshot, request.sketch_index)?;
                snapshot
                    .explode_sketch_block(sketch, SketchEntityId(request.instance_id), reservations)
                    .map_err(api_error)
            },
            |_, report| {
                encode(&ExplodeSketchBlockResult {
                    entity_refs: Some(report.copies.into_iter().map(|id| id.0).collect()),
                })
            },
        )
        .map_err(api_error)
}
