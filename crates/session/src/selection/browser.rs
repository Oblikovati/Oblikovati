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
