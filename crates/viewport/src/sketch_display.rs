// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchPlane},
};
/// Owned committed geometry. The document keeps identity; native samples remain disposable.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchDisplay {
    pub sketch: DocumentSketchId,
    pub plane: SketchPlane,
    pub geometry: Sketch2D,
}
/// Display failure names the sketch as well as its entity within the current source packet.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchDecline {
    pub sketch: DocumentSketchId,
    pub entity: oblikovati_model::sketch::SketchEntityId,
    pub cause: String,
}

/// Transient authored identities selected by the active document owner.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct ViewportSketchSelection {
    pub sketches: Vec<DocumentSketchId>,
    pub entities: Vec<(DocumentSketchId, oblikovati_model::sketch::SketchEntityId)>,
}
impl ViewportSketchSelection {
    pub(crate) fn normalized(&self) -> Self {
        let mut selection = self.clone();
        selection.sketches.sort_by_key(|id| id.0);
        selection.sketches.dedup();
        selection
            .entities
            .sort_by_key(|(sketch, entity)| (sketch.0, entity.0));
        selection.entities.dedup();
        selection
    }
}
