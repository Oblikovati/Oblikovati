// SPDX-License-Identifier: GPL-2.0-only
//! Placing a sketch point: its role and any coincidence are part of the same undo step.
use super::{
    DocumentSnapshot, RecipeReservations, SketchLineCreationError,
    sketch_primitive::append_primitive,
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlanarConstraint, SketchEntityId, SketchGeometricConstraintCreation,
        SketchPrimitiveCreation,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;

/// Where the point goes, whether it is a center point, and an existing point it lands on.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPointPlacement {
    pub position: [f64; 2],
    pub center: bool,
    pub coincident_with: Option<SketchEntityId>,
}
impl DocumentSnapshot {
    /// Author one point against current solved geometry.
    /// Example: `snapshot.create_sketch_point(sketch, &placement, options, ids)?;`.
    pub fn create_sketch_point(
        &mut self,
        sketch: DocumentSketchId,
        placement: &SketchPointPlacement,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchEntityId, SketchLineCreationError> {
        let mut geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let primitive = SketchPrimitiveCreation::Point {
            position: placement.position,
        };
        let point = append_primitive(&mut geometry, sketch, &primitive, ids)
            .map_err(SketchLineCreationError::Primitive)?
            .entity;
        geometry
            .set_center_point(point, placement.center)
            .map_err(SketchLineCreationError::Sketch)?;
        if let Some(other) = placement.coincident_with {
            let constraint = ids
                .allocate_sketch_constraint(sketch)
                .map_err(SketchLineCreationError::Constraint)?;
            SketchGeometricConstraintCreation::Relation(PlanarConstraint::Coincident {
                points: [other, point],
            })
            .append(&mut geometry, constraint)
            .map_err(SketchLineCreationError::Sketch)?;
        }
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok(point)
    }
}
