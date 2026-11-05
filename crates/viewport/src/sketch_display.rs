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
