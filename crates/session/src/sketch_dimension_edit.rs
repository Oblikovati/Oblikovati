// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::SketchDimensionEdit,
    sketch::{SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Edit a dimension and recompute current driving sources through history.
    /// Example: `session.edit_sketch_dimension(sketch, dimension, &edit)?;`.
    pub fn edit_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), CommandError<SketchError>> {
        self.edit_sketch_dimension_projected(sketch, dimension, edit, |reply| reply)
    }
    /// Prepare the response before candidate adoption; rejected replies preserve all dimension state.
    /// Example: `session.edit_sketch_dimension_projected(sketch, dimension, &edit, |reply| encode(reply?))?;`.
    pub fn edit_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
        project: impl FnOnce(Result<(), SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Edit sketch dimension", |snapshot, _| {
            project(snapshot.edit_sketch_dimension(sketch, dimension, edit))
        })
    }
}
