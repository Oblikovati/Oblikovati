// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_primitive::{append_defining_positions, append_primitive};
use super::{DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationError};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId, SketchPathMember, SketchSlotLayout, SketchSlotPlacement},
};
/// One slot placement with a positive width and a role for all boundary members.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchSlotCreation {
    pub placement: SketchSlotPlacement,
    pub width: f64,
    pub construction: bool,
}
/// Ordered boundary identities and all owned seed points, including hidden centers.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchSlotCreationReport {
    pub sketch: DocumentSketchId,
    pub members: Vec<SketchEntityId>,
    pub points: Vec<SketchEntityId>,
    pub exposed_points: Vec<SketchEntityId>,
}
impl DocumentSnapshot {
    /// Adopt the complete slot on a copy; example: `snapshot.create_sketch_slot(sketch, &input, ids)?`.
    pub fn create_sketch_slot(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchSlotCreation,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchSlotCreationReport, SketchPrimitiveCreationError> {
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(SketchPrimitiveCreationError::Sketch)?
            .geometry()
            .clone();
        let layout = input
            .placement
            .layout(input.width)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        let report = append_slot(&mut geometry, sketch, input, &layout, ids)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        Ok(report)
    }
}
fn append_slot(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    input: &SketchSlotCreation,
    layout: &SketchSlotLayout,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchSlotCreationReport, SketchPrimitiveCreationError> {
    let points = append_defining_positions(geometry, &layout.points, ids)?;
    let mut members = Vec::new();
    let mut exposed_points = Vec::new();
    for member in layout.members {
        let primitive = member
            .primitive(&points, input.construction)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        members.push(append_primitive(geometry, sketch, &primitive, ids)?.entity);
        if let SketchPathMember::Line(pair) = member {
            exposed_points.extend(pair.map(|index| points[index]));
        }
    }
    Ok(SketchSlotCreationReport {
        sketch,
        members,
        points,
        exposed_points,
    })
}
