// SPDX-License-Identifier: GPL-2.0-only
//! Route constraint glyph visibility through the sketch history owner.
use super::*;
use oblikovati_model::sketch::SketchConstraintId;

pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: ConstraintGlyphsVisibilityArgs,
) -> Result<Vec<u8>, Error> {
    let (document, id) = active_sketch(files, request.sketch_index)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let sketch = session.snapshot().sketches.get(id).map_err(api_error)?;
    let geometry = sketch.geometry();
    let mut relation_ids: Vec<_> = geometry.geometric_relation_ids().map(|id| id.0).collect();
    relation_ids.sort_unstable();
    encode(&ConstraintGlyphsVisibilityResult {
        shown: sketch.constraints_shown(),
        relation_i_ds: Some(relation_ids),
        hidden_i_ds: Some(
            sketch
                .hidden_constraint_glyphs()
                .iter()
                .map(|id| id.0)
                .collect(),
        ),
    })
}

pub(super) fn set_one<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: ConstraintGlyphVisibilityArgs,
) -> Result<Vec<u8>, Error> {
    let (document, id) = active_sketch(files, request.sketch_index)?;
    let reply = encode(&OKResult { ok: true })?;
    files
        .commands(document)
        .map_err(api_error)?
        .set_constraint_glyph_visible(
            id,
            SketchConstraintId(request.constraint_id),
            request.visible,
        )
        .map_err(api_error)?;
    Ok(reply)
}

pub(super) fn set_all<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AllConstraintGlyphsVisibilityArgs,
) -> Result<Vec<u8>, Error> {
    let (document, id) = active_sketch(files, request.sketch_index)?;
    let reply = encode(&OKResult { ok: true })?;
    files
        .commands(document)
        .map_err(api_error)?
        .set_sketch_constraints_shown(id, request.visible)
        .map_err(api_error)?;
    Ok(reply)
}
