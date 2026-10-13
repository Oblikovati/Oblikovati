// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_primitive::{append_defining_positions, append_primitive};
use super::{DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationError};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        Sketch2D, SketchEntityId, SketchError, SketchPointInput, SketchPrimitiveCreation,
        require_position,
    },
};
/// Seed vertices in authored order; closure shares the first point's identity.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPolylineCreation {
    pub points: Vec<[f64; 2]>,
    pub closed: bool,
    pub construction: bool,
}
/// Ordered line IDs and unique defining point IDs; no geometric profile is certified.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPolylineCreationReport {
    pub sketch: DocumentSketchId,
    pub lines: Vec<SketchEntityId>,
    pub points: Vec<SketchEntityId>,
}
impl SketchPolylineCreation {
    fn validate(&self) -> Result<(), SketchError> {
        let minimum = if self.closed { 3 } else { 2 };
        if self.points.len() < minimum {
            return Err(SketchError::Definition(format!(
                "polyline vertices {:?}, closed {}: expected at least {minimum} finite positions",
                self.points, self.closed
            )));
        }
        for point in &self.points {
            require_position(*point)?;
        }
        Ok(())
    }
}
impl DocumentSnapshot {
    /// Create an atomic chain with shared hidden vertices and no implicit constraints.
    /// Example: `snapshot.create_sketch_polyline(sketch, &input, ids)?;`.
    pub fn create_sketch_polyline(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolylineCreation,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchPolylineCreationReport, SketchPrimitiveCreationError> {
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(SketchPrimitiveCreationError::Sketch)?
            .geometry()
            .clone();
        let report = append_polyline(&mut geometry, sketch, input, ids)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        Ok(report)
    }
}
pub(super) fn append_polyline(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    input: &SketchPolylineCreation,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchPolylineCreationReport, SketchPrimitiveCreationError> {
    input
        .validate()
        .map_err(SketchPrimitiveCreationError::Sketch)?;
    let points = append_defining_positions(geometry, &input.points, ids)?;
    let lines = append_polyline_lines(geometry, sketch, input, &points, ids)?;
    Ok(SketchPolylineCreationReport {
        sketch,
        lines,
        points,
    })
}
fn append_polyline_lines(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    input: &SketchPolylineCreation,
    points: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchEntityId>, SketchPrimitiveCreationError> {
    let count = if input.closed {
        points.len()
    } else {
        points.len() - 1
    };
    let mut lines = Vec::with_capacity(count);
    for index in 0..count {
        let line = SketchPrimitiveCreation::Line {
            endpoints: [points[index], points[(index + 1) % points.len()]]
                .map(SketchPointInput::Owned),
            construction: input.construction,
            centerline: false,
        };
        lines.push(append_primitive(geometry, sketch, &line, ids)?.entity);
    }
    Ok(lines)
}
