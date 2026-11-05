// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    feature::{FeatureId, sketch_prism::SketchPrismDefinition},
    history::HistoryError,
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Create extrusion inputs through the shared transaction and recompute boundary.
    /// Example: `session.create_sketch_prism("Extrude", definition)?`.
    pub fn create_sketch_prism(
        &mut self,
        name: &str,
        definition: SketchPrismDefinition,
    ) -> Result<FeatureId, CommandError<HistoryError>> {
        self.create_sketch_prism_projected(name, definition, |result| result)
    }
    /// Prepare a reply before adopting extrusion inputs.
    /// Example: `session.create_sketch_prism_projected("Extrude", definition, encode)?`.
    pub fn create_sketch_prism_projected<R, E>(
        &mut self,
        name: &str,
        definition: SketchPrismDefinition,
        project: impl FnOnce(Result<FeatureId, HistoryError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Create extrusion", |snapshot, ids| {
            project(snapshot.create_sketch_prism(name, definition, ids))
        })
    }
}
