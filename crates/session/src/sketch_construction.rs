// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchConstructionReport},
    sketch::{SketchEntityId, SketchError},
};
/// Borrow checked metadata evidence before history adoption.
/// Example: `let reply: SketchConstructionProjectionInput<'_> = Ok(&report);`.
pub type SketchConstructionProjectionInput<'report> =
    Result<&'report SketchConstructionReport, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Edit curve roles through normal history without recomputing unchanged coordinates.
    /// Example: `session.set_sketch_construction(sketch, &[line], true)?;`.
    pub fn set_sketch_construction(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchConstructionReport, CommandError<SketchError>> {
        self.set_sketch_construction_projected(sketch, entities, construction, |report| {
            report.cloned()
        })
    }
    /// Prepare a reply before adopting construction flags; reply errors discard the candidate.
    /// Example: `session.set_sketch_construction_projected(sketch, &[line], true, |report| encode(report?))?;`.
    pub fn set_sketch_construction_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        construction: bool,
        project: impl FnOnce(SketchConstructionProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run_with_recompute(
            "Set sketch construction",
            |snapshot, _| {
                let report = snapshot
                    .sketches
                    .set_construction(sketch, entities, construction);
                project(report.as_ref().map_err(Clone::clone))
            },
            SketchUpdateMode::Explicit,
        )
    }
}
