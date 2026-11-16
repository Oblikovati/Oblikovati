// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchBlockInstance},
    sketch::{Sketch2D, SketchPlane},
};
/// Owned committed geometry. The document keeps identity; native samples remain disposable.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchDisplay {
    pub sketch: DocumentSketchId,
    pub plane: SketchPlane,
    pub geometry: Sketch2D,
    /// Definition geometry and placement for each visible block instance.
    pub blocks: Vec<ViewportSketchBlock>,
    /// What its lines draw in, in place of the palette's geometry colour.
    pub line_color: Option<oblikovati_model::theme::Rgba>,
}
/// One definition rendered through its owning instance's placement.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchBlock {
    pub instance: SketchBlockInstance,
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
/// Transient candidate entities; example: a hovered parallel-constraint glyph.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ViewportSketchHover {
    pub sketch: DocumentSketchId,
    pub entities: Vec<oblikovati_model::sketch::SketchEntityId>,
}
/// Transient feedback for authored entities or a derived point, such as a midpoint.
#[derive(Clone, Debug, PartialEq)]
pub enum ViewportSketchFeedback {
    Entities(ViewportSketchHover),
    Point {
        sketch: DocumentSketchId,
        position: [f64; 2],
    },
}
// #3795: only the Linux renderer consumes normalization; pure selection checks remain portable.
#[cfg(any(target_os = "linux", test))]
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

#[cfg(test)]
#[path = "../tests/sketch_selection.rs"]
mod tests;
