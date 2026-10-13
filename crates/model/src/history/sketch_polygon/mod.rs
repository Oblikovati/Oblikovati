// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, SketchPolylineCreation, SketchPolylineCreationReport,
    sketch_polyline::append_polyline, sketch_primitive::append_primitive,
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        Sketch2D, SketchConstraintId, SketchEntityId, SketchPointInput, SketchPolygonSeed,
        SketchPrimitiveCreation,
    },
};
mod error;
mod relations;
pub use error::SketchPolygonCreationError;
/// Regular polygon input; the construction circle and regularity relations are mandatory.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPolygonCreation {
    pub seed: SketchPolygonSeed,
    pub construction: bool,
}
/// Permanent chain, center, construction circle, and local constraint identities.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPolygonCreationReport {
    pub chain: SketchPolylineCreationReport,
    pub center: SketchEntityId,
    pub circle: SketchEntityId,
    pub constraints: Vec<SketchConstraintId>,
}
impl DocumentSnapshot {
    /// Create one atomic regular polygon with its defining circle and constraints.
    /// Example: `snapshot.create_sketch_polygon(sketch, &input, ids)?` inside an edit.
    pub fn create_sketch_polygon(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolygonCreation,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchPolygonCreationReport, SketchPolygonCreationError> {
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(SketchPolygonCreationError::Sketch)?
            .geometry()
            .clone();
        let (chain_input, circle_input) = input.prepare(&geometry)?;
        let chain = append_polyline(&mut geometry, sketch, &chain_input, ids)
            .map_err(SketchPolygonCreationError::Primitive)?;
        let circle = append_primitive(&mut geometry, sketch, &circle_input, ids)
            .map_err(SketchPolygonCreationError::Primitive)?;
        let constraints = relations::append(&mut geometry, &chain, circle.entity, ids)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchPolygonCreationError::Sketch)?;
        Ok(SketchPolygonCreationReport {
            chain,
            center: circle.points[0],
            circle: circle.entity,
            constraints,
        })
    }
}
impl SketchPolygonCreation {
    fn prepare(
        self,
        geometry: &Sketch2D,
    ) -> Result<(SketchPolylineCreation, SketchPrimitiveCreation), SketchPolygonCreationError> {
        let points = self
            .seed
            .vertices()
            .map_err(SketchPolygonCreationError::Sketch)?;
        let offset = [
            points[0][0] - self.seed.center[0],
            points[0][1] - self.seed.center[1],
        ];
        // Go derives the construction radius from the first authored vertex (#3563, aa8297d1).
        let radius = (offset[0] * offset[0] + offset[1] * offset[1]).sqrt();
        let circle = SketchPrimitiveCreation::Circle {
            center: SketchPointInput::New(self.seed.center),
            radius,
            construction: true,
        };
        circle
            .validate(geometry)
            .map_err(SketchPolygonCreationError::Sketch)?;
        Ok((
            SketchPolylineCreation {
                points,
                closed: true,
                construction: self.construction,
            },
            circle,
        ))
    }
}
