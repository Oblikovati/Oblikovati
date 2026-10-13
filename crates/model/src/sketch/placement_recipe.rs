// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, SketchPathMember, require_position};
mod bindings;
mod fields;
mod rectangles;
mod slots;
pub use bindings::{SketchPlacementBinding, SketchPlacementRelation};
/// A shared authoring graph for preview and atomic commit; indices are local, not permanent IDs.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementRecipe {
    pub points: Vec<[f64; 2]>,
    pub members: Vec<SketchPlacementMember>,
    pub relations: Vec<SketchPlacementRelation>,
    pub fields: Vec<SketchPlacementField>,
}
/// One boundary or construction member using the recipe's point indices.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPlacementMember {
    pub path: SketchPathMember,
    pub construction: bool,
}
/// A measured placement field; steering fields do not create a dimension on commit.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementField {
    pub label: String,
    pub value: f64,
    pub witness: [[f64; 2]; 2],
    pub binding: SketchPlacementBinding,
}
impl SketchPlacementRecipe {
    pub(crate) fn validate(&self) -> Result<(), SketchError> {
        for point in &self.points {
            require_position(*point)?;
        }
        for field in &self.fields {
            require_position(field.witness[0])?;
            require_position(field.witness[1])?;
            if !field.value.is_finite() {
                return Err(SketchError::Definition(format!(
                    "placement field {:?}, value {}: expected a finite measurement",
                    field.label, field.value
                )));
            }
        }
        Ok(())
    }
}
