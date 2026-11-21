// SPDX-License-Identifier: GPL-2.0-only
//! Authored revolve inputs, separate from kernel results and complete feature replay.
mod error;
mod recipe;
mod resolve;
mod validation;
use crate::{
    document_sketch::DocumentSketchId,
    feature::{sketch_prism::SketchPrismSelection, work_geometry::DatumAngle},
};
pub use error::{SketchRevolveCause, SketchRevolveError};
pub use recipe::SketchRevolveRecipe;
pub use resolve::ResolvedSketchRevolve;

/// Validated authored revolve input with one implicit current sketch centerline axis.
#[derive(Clone, Debug)]
pub struct SketchRevolveDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    angle: DatumAngle,
}

impl SketchRevolveDefinition {
    /// Validate authored profile identities and one angular driver.
    /// Example: `SketchRevolveDefinition::new(sketch, selection, DatumAngle::parameter(3))?`.
    pub fn new(
        sketch: DocumentSketchId,
        selection: SketchPrismSelection,
        angle: DatumAngle,
    ) -> Result<Self, SketchRevolveError> {
        let definition = Self {
            sketch,
            selection,
            angle,
        };
        validation::validate_sources(&definition).map_err(|cause| definition.decline(cause))?;
        Ok(definition)
    }

    /// Inspect the stable source sketch; example: `definition.sketch()` before dependency analysis.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }

    /// Inspect explicit profile traversal; example: `definition.selection().outer`.
    pub fn selection(&self) -> &SketchPrismSelection {
        &self.selection
    }

    /// Inspect the retained angular driver; example: `definition.angle()`.
    pub fn angle(&self) -> DatumAngle {
        self.angle
    }

    /// Compare only authored sources, including exact scalar representation.
    /// Example: `before.same_recipe(&after)` before a no-op edit is accepted.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.sketch == other.sketch
            && self.selection == other.selection
            && self.angle.same_recipe(other.angle)
    }

    fn decline(&self, cause: SketchRevolveCause) -> SketchRevolveError {
        SketchRevolveError {
            sketch: self.sketch,
            cause: Box::new(cause),
        }
    }
}

impl PartialEq for SketchRevolveDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
