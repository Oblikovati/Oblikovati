// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraint, SketchConstraintId, SketchError},
};

impl DocumentSnapshot {
    /// Delete by stable identity on isolated geometry; failed sketches can remove faulty relations.
    /// Example: `snapshot.delete_sketch_geometric_constraint(sketch, constraint)?;` inside an edit.
    pub fn delete_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<SketchConstraint, SketchError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        let removed = geometry.remove_geometric_constraint(constraint)?;
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok(removed)
    }
}
