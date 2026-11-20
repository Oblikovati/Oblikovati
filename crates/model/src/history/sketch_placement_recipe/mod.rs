// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, SketchDimensionCreationReport,
    sketch_primitive::{append_defining_positions, append_primitive},
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlacementPrimitive, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
        SketchInferenceCandidates, SketchPlacementRecipe,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;
mod error;
mod fields;
pub(super) mod inference;
mod picked;
pub use error::SketchPlacementCreationError;
/// A user-entered expression and the preview's optional annotation position, in working units.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementFieldLock {
    pub reference: bool,
    pub expression: String,
    pub text_point: Option<[f64; 2]>,
}
/// Permanent identities from one atomic recipe; field reports retain their input index.
#[derive(Clone, Debug)]
pub struct SketchPlacementCreationReport {
    pub sketch: DocumentSketchId,
    pub points: Vec<SketchEntityId>,
    pub members: Vec<SketchEntityId>,
    pub relations: Vec<SketchConstraintId>,
    pub dimensions: Vec<(usize, SketchDimensionCreationReport)>,
    inference_witness: Option<inference::TransientPlacement>,
}
// The witness validates publication; it is not part of the authored creation report.
impl PartialEq for SketchPlacementCreationReport {
    fn eq(&self, other: &Self) -> bool {
        self.sketch == other.sketch
            && self.points == other.points
            && self.members == other.members
            && self.relations == other.relations
            && self.dimensions == other.dimensions
    }
}
impl SketchPlacementCreationReport {
    /// Check that a later placement solve kept the source and inferred anchors fixed.
    /// Example: `report.verify_inference_preserved(&geometry)?` before publishing a candidate.
    pub fn verify_inference_preserved(
        &self,
        geometry: &Sketch2D,
    ) -> Result<(), crate::sketch::SketchTransientSolveError> {
        match &self.inference_witness {
            Some(witness) => witness.check_preserved(geometry),
            None => Ok(()),
        }
    }
}
/// Evaluated inputs that govern placement inference; example: `SketchPlacementContext { datums: None, scope: Some(&scope) }`.
#[derive(Clone, Copy)]
pub struct SketchPlacementContext<'a> {
    pub datums: Option<&'a crate::feature::work_geometry::WorkGeometryEvaluation>,
    pub scope: Option<&'a SketchInferenceCandidates>,
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
        self.create_sketch_placement_from(
            sketch,
            recipe,
            locks,
            options,
            ids,
            SketchPlacementContext {
                datums: None,
                scope: None,
            },
        )
    }
    /// Create a placement using one current part replay.
    /// Example: `snapshot.create_sketch_placement_with_datums(sketch, recipe, locks, options, ids, datums)?`.
    pub fn create_sketch_placement_with_datums(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        datums: &crate::feature::work_geometry::WorkGeometryEvaluation,
    ) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
        self.create_sketch_placement_from(
            sketch,
            recipe,
            locks,
            options,
            ids,
            SketchPlacementContext {
                datums: Some(datums),
                scope: None,
            },
        )
    }
    /// Create with a session inference scope and optional evaluated datums.
    /// Example: `snapshot.create_scoped_sketch_placement(sketch, recipe, locks, options, ids, context)?`.
    pub fn create_scoped_sketch_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        context: SketchPlacementContext<'_>,
    ) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
        self.create_sketch_placement_from(sketch, recipe, locks, options, ids, context)
    }
    fn create_sketch_placement_from(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        context: SketchPlacementContext<'_>,
    ) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
        let SketchPlacementContext { datums, scope } = context;
        recipe.validate()?;
        let mut candidate = self.clone();
        let (mut report, transient) =
            candidate.append_placement(sketch, recipe, options, ids, datums, scope)?;
        fields::append(
            &mut candidate,
            recipe,
            locks,
            &mut report,
            options,
            ids,
            inference::PlacementSolveContext {
                datums,
                transient: transient.as_ref(),
            },
        )?;
        report.inference_witness = transient;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(report)
    }
    fn append_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        datums: Option<&crate::feature::work_geometry::WorkGeometryEvaluation>,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<
        (
            SketchPlacementCreationReport,
            Option<inference::TransientPlacement>,
        ),
        SketchPlacementCreationError,
    > {
        let mut geometry = self.current_sketch_authoring_geometry_from(sketch, options, datums)?;
        let settings = self.effective_sketch_settings();
        let source = if !recipe.picked_points.is_empty()
            || (settings.infer_constraints && settings.auto_apply_constraints)
        {
            Some(crate::sketch::SketchInferenceFreeze::capture(&geometry)?)
        } else {
            None
        };
        let (points, fresh) = picked::append_points(&mut geometry, recipe, ids)?;
        // Primitives can add auxiliary points; infer before those can become fresh targets.
        let inferred = inference::append(
            self,
            &mut geometry,
            sketch,
            fresh.as_deref().unwrap_or(&points),
            ids,
            scope,
        )?;
        let members = append_members(&mut geometry, sketch, recipe, &points, ids)?;
        let mut relations =
            append_relations(&mut geometry, sketch, recipe, &points, &members, ids)?;
        let has_inference = !recipe.picked_points.is_empty()
            || !inferred.relations.is_empty()
            || !inferred.anchors.is_empty();
        relations.extend(inferred.relations);
        let transient =
            source
                .filter(|_| has_inference)
                .map(|source| inference::TransientPlacement {
                    source,
                    anchors: inferred.anchors,
                    relations: Vec::new(),
                });
        if let Some(context) = &transient {
            geometry = context
                .solve(self, &geometry, sketch, options, datums)
                .map_err(SketchPlacementCreationError::Inference)?;
        }
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok((
            SketchPlacementCreationReport {
                sketch,
                points,
                members,
                relations,
                dimensions: Vec::new(),
                inference_witness: None,
            },
            transient,
        ))
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
            let primitive =
                match member
                    .geometry
                    .primitive(points, member.construction, member.centerline)?
                {
                    PlacementPrimitive::Point(id) => {
                        geometry.expose_defining_point(id)?;
                        return Ok(id);
                    }
                    PlacementPrimitive::Curve(primitive) => primitive,
                };
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
    points: &[SketchEntityId],
    members: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchPlacementCreationError> {
    recipe
        .relations
        .iter()
        .map(|relation| {
            let input = relation.resolve(points, members)?;
            input.validate(geometry)?;
            let id = ids
                .allocate_sketch_constraint(sketch)
                .map_err(SketchPlacementCreationError::Constraint)?;
            input.append(geometry, id)?;
            Ok(id)
        })
        .collect()
}
