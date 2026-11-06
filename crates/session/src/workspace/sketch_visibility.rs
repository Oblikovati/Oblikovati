// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchError,
};

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Change one owned sketch's display state; example: `commands.set_sketch_visibility(id, false)?`.
    pub fn set_sketch_visibility(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.session.set_sketch_visibility(sketch, visible)
    }
    /// Validate the metadata reply before adoption; example: `commands.set_sketch_visibility_projected(id, false, |reply| encode(reply?))?`.
    pub fn set_sketch_visibility_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
        project: impl FnOnce(Result<&DocumentSnapshot, SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .set_sketch_visibility_projected(sketch, visible, project)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Show or hide the relation marks; example: `commands.set_sketch_constraints_shown(id, true)?`.
    pub fn set_sketch_constraints_shown(
        &mut self,
        sketch: DocumentSketchId,
        shown: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.session.set_sketch_constraints_shown(sketch, shown)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Name a design view of this document; example: `commands.create_design_view("Front")`.
    pub fn create_design_view(&mut self, name: &str) -> Result<(), CommandError<SketchError>> {
        self.session.create_design_view(name)
    }
}
