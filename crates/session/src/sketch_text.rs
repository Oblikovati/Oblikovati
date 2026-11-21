// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_api::types::SketchTextStyle;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::SketchPrimitiveCreationError,
    sketch::{SketchEntityId, SketchError},
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Create one undoable text entity, for example `session.create_sketch_text(sketch, point, style)?`.
    pub fn create_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        position: [f64; 2],
        style: SketchTextStyle,
    ) -> Result<SketchEntityId, CommandError<SketchPrimitiveCreationError>> {
        self.run("Create sketch text", |snapshot, ids| {
            snapshot.create_sketch_text(sketch, position, style, ids)
        })
    }

    /// Change the stored text in one history step, for example `session.edit_sketch_text(sketch, id, style)?`.
    pub fn edit_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        id: SketchEntityId,
        style: SketchTextStyle,
    ) -> Result<(), CommandError<SketchError>> {
        self.run("Edit sketch text", |snapshot, _| {
            snapshot.edit_sketch_text(sketch, id, style)
        })
    }
}
