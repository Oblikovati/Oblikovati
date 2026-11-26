// SPDX-License-Identifier: GPL-2.0-only
//! Authored prism inputs, separate from kernel results and the complete extrude feature contract.
mod cut;
mod error;
mod extent;
mod recipe;
mod resolve;
pub(crate) mod validation;
use super::work_geometry::DatumLength;
use crate::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
pub use cut::SketchPrismCutTarget;
pub use error::{SketchPrismCause, SketchPrismError};
pub use extent::SketchPrismExtent;
pub use recipe::SketchPrismRecipe;
pub use resolve::ResolvedSketchPrism;
/// One authored curve in a directed boundary; reversal changes traversal, not its source identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDirectedCurve {
    #[serde(alias = "line")]
    pub curve: SketchEntityId,
    pub reversed: bool,
}
/// Explicit ordered boundaries over shared authored endpoints; this is not automatic region detection.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPrismSelection {
    pub outer: Vec<SketchDirectedCurve>,
    pub holes: Vec<Vec<SketchDirectedCurve>>,
}
/// Validated authored input. Evaluated coordinates, solve reports and native shapes are not stored here.
#[derive(Clone, Debug)]
pub struct SketchPrismDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    displacement: [DatumLength; 3],
    direction: SketchPrismDirection,
    second_depth: Option<DatumLength>,
    extent: Option<SketchPrismExtent>,
    cut_target: Option<SketchPrismCutTarget>,
}
/// Placement of the extrusion span relative to the source sketch plane.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum SketchPrismDirection {
    #[default]
    Positive,
    Negative,
    Symmetric,
    Asymmetric,
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
            direction: SketchPrismDirection::Positive,
            second_depth: None,
            extent: None,
            cut_target: None,
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
    /// Set how a positive authored depth is placed; example: `definition.with_direction(SketchPrismDirection::Symmetric)?`.
    pub fn with_direction(
        mut self,
        direction: SketchPrismDirection,
    ) -> Result<Self, SketchPrismError> {
        self.direction = direction;
        validation::validate_sources(&self).map_err(|cause| self.decline(cause))?;
        Ok(self)
    }
    /// Read the span placement; example: `definition.direction()` when opening Edit Feature.
    pub fn direction(&self) -> SketchPrismDirection {
        self.direction
    }
    /// Read Distance B when present; example: `definition.second_depth()` for the edit panel.
    pub fn second_depth(&self) -> Option<DatumLength> {
        self.second_depth
    }
    /// Return the authored plane bounds, if any. Example: `definition.extent()` before exposing distance edits.
    pub fn extent(&self) -> Option<&SketchPrismExtent> {
        self.extent.as_ref()
    }
    /// Set the opposite-side depth; example: `definition.with_second_depth(DatumLength::Constant(0.5))?`.
    pub fn with_second_depth(mut self, depth: DatumLength) -> Result<Self, SketchPrismError> {
        self.second_depth = Some(depth);
        self.with_direction(SketchPrismDirection::Asymmetric)
    }
    /// Replace the authored depth without changing its direction.
    /// Example: `definition.with_normal_source(DatumLength::Parameter(3))?`.
    pub fn with_normal_source(&self, depth: DatumLength) -> Result<Self, SketchPrismError> {
        let mut next = self.clone();
        next.displacement[2] = depth;
        validation::validate_sources(&next).map_err(|cause| next.decline(cause))?;
        Ok(next)
    }
    /// Compare exact authored values, including signed zero; example: `before.same_recipe(&after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.sketch == other.sketch
            && self.selection == other.selection
            && self.direction == other.direction
            && self.extent == other.extent
            && self.cut_target == other.cut_target
            && match (self.second_depth, other.second_depth) {
                (None, None) => true,
                (Some(a), Some(b)) => a.same_recipe(b),
                _ => false,
            }
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

impl PartialEq for SketchPrismDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
