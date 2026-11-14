// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchPlane},
};
pub(super) fn owned_sketch_session(
    veto: Option<&'static str>,
    name: &str,
    geometry: Sketch2D,
) -> DocumentSession<RecordingObserver> {
    let mut snapshot = recipe();
    snapshot
        .sketches
        .add(DocumentSketchId(1), name, SketchPlane::xy(), geometry)
        .unwrap();
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    DocumentSession::new(
        17,
        history,
        RecordingObserver {
            veto,
            ..Default::default()
        },
    )
}
