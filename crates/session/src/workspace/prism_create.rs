// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    feature::{FeatureId, sketch_prism::SketchPrismDefinition},
    history::HistoryError,
};

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create extrusion inputs on the selected document without changing focus.
    /// Example: `commands.create_sketch_prism("Extrude", definition)?`.
    pub fn create_sketch_prism(
        &mut self,
        name: &str,
        definition: SketchPrismDefinition,
    ) -> Result<FeatureId, CommandError<HistoryError>> {
        self.session.create_sketch_prism(name, definition)
    }
    /// Prepare the API reply before adopting the document's extrusion.
    /// Example: `commands.create_sketch_prism_projected("Extrude", definition, encode)?`.
    pub fn create_sketch_prism_projected<R, E>(
        &mut self,
        name: &str,
        definition: SketchPrismDefinition,
        project: impl FnOnce(Result<FeatureId, HistoryError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_sketch_prism_projected(name, definition, project)
    }
}
