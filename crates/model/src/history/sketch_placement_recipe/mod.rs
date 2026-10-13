// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, SketchDimensionCreationReport,
    sketch_primitive::{append_defining_positions, append_primitive},
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchConstraintId, SketchEntityId, SketchPlacementRecipe},
};
use oblikovati_solve::nonlinear::SolveOptions;
mod error;
mod fields;
pub use error::SketchPlacementCreationError;
/// A user-entered expression and the preview's optional annotation position, in working units.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementFieldLock {
    pub expression: String,
    pub text_point: Option<[f64; 2]>,
}
/// Permanent identities from one atomic recipe; field reports retain their input index.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementCreationReport {
    pub sketch: DocumentSketchId,
    pub points: Vec<SketchEntityId>,
    pub members: Vec<SketchEntityId>,
    pub relations: Vec<SketchConstraintId>,
    pub dimensions: Vec<(usize, SketchDimensionCreationReport)>,
}
impl DocumentSnapshot {
    /// Apply geometry, relations, and locked dimensions as one candidate; example: `snapshot.create_sketch_placement(sketch,&recipe,&locks,options,ids)?`.
    pub fn create_sketch_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
        recipe.validate()?;
        let mut candidate = self.clone();
        let mut report = candidate.append_placement(sketch, recipe, ids)?;
        fields::append(&mut candidate, recipe, locks, &mut report, options, ids)?;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(report)
    }
    fn append_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        let points = append_defining_positions(&mut geometry, &recipe.points, ids)
            .map_err(SketchPlacementCreationError::Primitive)?;
        let members = append_members(&mut geometry, sketch, recipe, &points, ids)?;
        let relations = append_relations(&mut geometry, sketch, recipe, &members, ids)?;
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok(SketchPlacementCreationReport {
            sketch,
            points,
            members,
            relations,
            dimensions: Vec::new(),
        })
    }
}
fn append_members(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    recipe: &SketchPlacementRecipe,
    points: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchEntityId>, SketchPlacementCreationError> {
    recipe
        .members
        .iter()
        .map(|member| {
            let primitive = member.path.primitive(points, member.construction)?;
            primitive.validate(geometry)?;
            append_primitive(geometry, sketch, &primitive, ids)
                .map(|report| report.entity)
                .map_err(SketchPlacementCreationError::Primitive)
        })
        .collect()
}
fn append_relations(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    recipe: &SketchPlacementRecipe,
    members: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchPlacementCreationError> {
    recipe
        .relations
        .iter()
        .map(|relation| {
            let input = relation.resolve(members)?;
            input.validate(geometry)?;
            let id = ids
                .allocate_sketch_constraint(sketch)
                .map_err(SketchPlacementCreationError::Constraint)?;
            input.append(geometry, id)?;
            Ok(id)
        })
        .collect()
}
