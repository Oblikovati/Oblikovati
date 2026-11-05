// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchConstructionProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchConstructionReport},
    sketch::{SketchEntityId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Edit this owner's curve roles without changing workspace focus.
    /// Example: `commands.set_sketch_construction(sketch, &[line], true)?;`.
    pub fn set_sketch_construction(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchConstructionReport, CommandError<SketchError>> {
        self.session
            .set_sketch_construction(sketch, entities, construction)
    }
    /// Prepare a typed reply before owner adoption.
    /// Example: `commands.set_sketch_construction_projected(sketch, &[line], true, |report| encode(report?))?;`.
    pub fn set_sketch_construction_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        construction: bool,
        project: impl FnOnce(SketchConstructionProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .set_sketch_construction_projected(sketch, entities, construction, project)
    }
}
