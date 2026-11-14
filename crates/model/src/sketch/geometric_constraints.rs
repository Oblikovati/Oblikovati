// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchConstraint, SketchConstraintId, SketchError, dependencies::dimension_parameter,
};
impl Sketch2D {
    /// Enumerate geometric relations in authored order, independently of dimensional sources.
    /// Example: `sketch.geometric_constraints().map(|constraint| constraint.id)` for API rows.
    pub fn geometric_constraints(&self) -> impl Iterator<Item = &SketchConstraint> {
        self.constraints
            .iter()
            .filter(|constraint| dimension_parameter(&constraint.relation).is_none())
    }
    /// Resolve an applied relation's geometric index; dimensional IDs and unknown IDs are errors.
    /// Example: `let index = sketch.geometric_constraint_index(created_constraint)?;`.
    pub fn geometric_constraint_index(&self, id: SketchConstraintId) -> Result<usize, SketchError> {
        self.geometric_constraints().position(|constraint| constraint.id == id).ok_or_else(|| {
            SketchError::Definition(format!("sketch constraint {id:?}: expected an owned geometric relation for geometric enumeration"))
        })
    }
    /// Delete one owned geometric relation without changing surviving identities or dimensions.
    /// Example: `sketch.remove_geometric_constraint(selected.id)?;` after resolving a wire index.
    pub fn remove_geometric_constraint(
        &mut self,
        id: SketchConstraintId,
    ) -> Result<SketchConstraint, SketchError> {
        self.remove_authored_relation(id, false)
    }
    pub(super) fn remove_authored_relation(
        &mut self,
        id: SketchConstraintId,
        dimensional: bool,
    ) -> Result<SketchConstraint, SketchError> {
        let position = self.constraints.iter().position(|constraint| {
            constraint.id == id
                && dimension_parameter(&constraint.relation).is_some() == dimensional
        });
        let kind = if dimensional {
            "dimensional"
        } else {
            "geometric"
        };
        let position = position.ok_or_else(|| {
            SketchError::Definition(format!(
                "sketch constraint {id:?}: expected an owned {kind} relation for deletion"
            ))
        })?;
        self.dimension_text_points.remove(&id);
        self.reference_dimensions.remove(&id);
        self.dimension_drive_limits.remove(&id);
        Ok(self.constraints.remove(position))
    }
}
