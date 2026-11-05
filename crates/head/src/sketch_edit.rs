// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};
/// The selected edit is transient; its placed plane is resolved from the current document.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct HeadSketchEdit {
    pub document: u64,
    pub sketch: DocumentSketchId,
    pub plane: SketchPlane,
}
