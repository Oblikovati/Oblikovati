// SPDX-License-Identifier: GPL-2.0-only
//! Authored prism inputs, separate from kernel results and the complete extrude feature contract.
mod error;
mod recipe;
mod resolve;
mod validation;
use super::work_geometry::DatumLength;
use crate::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
pub use error::{SketchPrismCause, SketchPrismError};
pub use recipe::SketchPrismRecipe;
pub use resolve::ResolvedSketchPrism;
/// One authored line in a directed boundary; reversal changes traversal, not its source identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDirectedLine {
    pub line: SketchEntityId,
    pub reversed: bool,
}
/// Explicit ordered boundaries over shared authored endpoints; this is not automatic region detection.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPrismSelection {
    pub outer: Vec<SketchDirectedLine>,
    pub holes: Vec<Vec<SketchDirectedLine>>,
}
/// Validated authored input. Evaluated coordinates, solve reports and native shapes are not stored here.
#[derive(Clone, Debug)]
pub struct SketchPrismDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    displacement: [DatumLength; 3],
}
impl SketchPrismDefinition {
    /// Validate authored identities and scalar sources before adoption.
    /// Example: `SketchPrismDefinition::new(sketch_id, selection, [DatumLength::Constant(0.), DatumLength::Constant(0.), DatumLength::Parameter(3)])?`.
    pub fn new(
        sketch: DocumentSketchId,
        selection: SketchPrismSelection,
        displacement: [DatumLength; 3],
    ) -> Result<Self, SketchPrismError> {
        let definition = Self {
            sketch,
            selection,
            displacement,
        };
        validation::validate_sources(&definition).map_err(|cause| definition.decline(cause))?;
        Ok(definition)
    }
    /// Inspect the stable source sketch; example: `definition.sketch()` for owner incidence queries.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }
    /// Inspect source traversal without derived coordinates; example: `definition.selection().outer`.
    pub fn selection(&self) -> &SketchPrismSelection {
        &self.selection
    }
    /// Inspect local X, Y and normal scalar sources; example: `definition.displacement()[2]`.
    pub fn displacement(&self) -> [DatumLength; 3] {
        self.displacement
    }
    /// Compare exact authored values, including signed zero; example: `before.same_recipe(&after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.sketch == other.sketch
            && self.selection == other.selection
            && self
                .displacement
                .into_iter()
                .zip(other.displacement)
                .all(|(a, b)| a.same_recipe(b))
    }
    fn decline(&self, cause: SketchPrismCause) -> SketchPrismError {
        SketchPrismError {
            sketch: self.sketch,
            cause: Box::new(cause),
        }
    }
}
