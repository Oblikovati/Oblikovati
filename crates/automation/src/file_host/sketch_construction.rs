// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{SketchCircleConstruction, SketchConstructionPolicy};
/// Borrowed geometry capability and explicit authoring policy; no kernel type reaches automation.
#[derive(Clone, Copy)]
pub(crate) struct SketchConstructionAccess<'a> {
    pub constructor: &'a dyn SketchCircleConstruction,
    pub policy: SketchConstructionPolicy,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Supply geometry construction on this owner; example: `host.with_sketch_construction(&backend,policy)`.
    pub fn with_sketch_construction(
        mut self,
        constructor: &'a dyn SketchCircleConstruction,
        policy: SketchConstructionPolicy,
    ) -> Self {
        self.context.get_mut().sketch_construction = Some(SketchConstructionAccess {
            constructor,
            policy,
        });
        self
    }
}
