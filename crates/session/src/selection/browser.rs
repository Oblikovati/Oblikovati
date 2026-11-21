// SPDX-License-Identifier: GPL-2.0-only
use super::{Selectable, SelectionKind};
use oblikovati_model::document_sketch::DocumentSketchId;

/// A whole-sketch browser identity; recipe edits do not change its identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct BrowserSketchPick {
    pub(crate) document: u64,
    pub(crate) sketch: DocumentSketchId,
}
impl BrowserSketchPick {
    /// Read the stable sketch identity; example: `let sketch = pick.sketch();`.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }
}
impl Selectable for BrowserSketchPick {
    fn selection_kind(&self) -> SelectionKind {
        SelectionKind::Sketch
    }
}

/// A selected body keeps its exact reference and its producing feature.
#[derive(Clone, Debug)]
pub struct BrowserBodyPick {
    pub key: String,
    pub feature: oblikovati_model::feature::FeatureId,
}
impl PartialEq for BrowserBodyPick {
    fn eq(&self, other: &Self) -> bool {
        self.key == other.key
    }
}
impl Eq for BrowserBodyPick {}
impl Selectable for BrowserBodyPick {
    fn selection_kind(&self) -> SelectionKind {
        SelectionKind::Body
    }
}
