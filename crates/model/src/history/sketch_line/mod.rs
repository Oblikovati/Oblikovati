// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationReport,
    sketch_primitive::append_primitive,
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        Sketch2D, SketchConstraintId, SketchDirectionInference, SketchInferenceCandidates,
        SketchInferencePolicy, SketchLinePlacementInference, SketchPointInput,
        SketchPrimitiveCreation,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;
mod chain;
mod error;
pub use chain::{SketchLineChainCreation, SketchLineChainCreationReport};
mod relations;
mod tangent_arc;
pub use error::SketchLineCreationError;
pub use tangent_arc::{SketchTangentArcCreation, SketchTangentArcReport};
/// Fresh endpoints in document working units; inference reads existing geometry before allocation.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchLineCreation {
    pub endpoints: [[f64; 2]; 2],
    pub construction: bool,
    pub centerline: bool,
}
/// One applied direction relation retains its identity and current geometric enumeration position.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchAppliedLineDirection {
    pub inference: SketchDirectionInference,
    pub constraint: SketchConstraintId,
    pub geometric_index: usize,
}
/// Proposals remain visible without constraining; applied records exist only for real relations.
#[derive(Clone, Debug)]
pub struct SketchLineCreationReport {
    pub primitive: SketchPrimitiveCreationReport,
    pub inference: SketchLinePlacementInference,
    pub point_constraints: Vec<SketchConstraintId>,
    pub direction: Option<SketchAppliedLineDirection>,
    inference_witness: Option<super::sketch_placement_recipe::inference::TransientPlacement>,
    needs_preparation: bool,
}
// Validation evidence is transient and is not an authored line report.
impl PartialEq for SketchLineCreationReport {
    fn eq(&self, other: &Self) -> bool {
        self.primitive == other.primitive
            && self.inference == other.inference
            && self.point_constraints == other.point_constraints
            && self.direction == other.direction
    }
}
impl SketchLineCreationReport {
    /// Solve line guidance on a private candidate while keeping source geometry fixed.
    /// Example: `report.prepare_inference(&mut snapshot, options)?` before recomputation.
    pub fn prepare_inference(
        &self,
        snapshot: &mut DocumentSnapshot,
        options: SolveOptions,
    ) -> Result<(), SketchLineCreationError> {
        if !self.needs_preparation {
            return Ok(());
        }
        let Some(witness) = &self.inference_witness else {
            return Ok(());
        };
        let geometry = snapshot
            .sketches
            .get(self.primitive.sketch)
            .map_err(SketchLineCreationError::Sketch)?
            .geometry();
        let solved = witness
            .solve(snapshot, geometry, self.primitive.sketch, options, None)
            .map_err(|cause| inference_failure(self.primitive.sketch, cause))?;
        snapshot
            .sketches
            .replace_geometry(self.primitive.sketch, solved)
            .map_err(SketchLineCreationError::Sketch)
    }
    /// Check that a later solve kept the source and temporary line guidance.
    /// Example: `report.verify_inference_preserved(&snapshot, options)?` before publication.
    pub fn verify_inference_preserved(
        &self,
        snapshot: &DocumentSnapshot,
        options: SolveOptions,
    ) -> Result<(), SketchLineCreationError> {
        let Some(witness) = &self.inference_witness else {
            return Ok(());
        };
        let geometry = snapshot
            .sketches
            .get(self.primitive.sketch)
            .map_err(SketchLineCreationError::Sketch)?
            .geometry();
        witness
            .check_preserved(geometry)
            .map_err(SketchLineCreationError::Inference)?;
        if witness.relations.is_empty() {
            return Ok(());
        }
        let solved = witness
            .solve(snapshot, geometry, self.primitive.sketch, options, None)
            .map_err(|cause| inference_failure(self.primitive.sketch, cause))?;
        if !geometry.same_recipe(&solved) {
            return Err(SketchLineCreationError::InferenceChanged(crate::sketch::SketchError::Definition(
                "temporary line inference changed during publication: expected the solved direction to remain satisfied".into(),
            )));
        }
        Ok(())
    }
}
impl DocumentSnapshot {
    /// Author a fresh inferred line with one adoption; failures retain reservations, not geometry.
    /// Example: `snapshot.create_inferred_sketch_line(sketch, &input, policy, options, ids)?;` inside an edit.
    pub fn create_inferred_sketch_line(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineCreationReport, SketchLineCreationError> {
        self.create_inferred_line_scoped(sketch, input, policy, solve_options, ids, None)
    }
    /// Create an inferred line using an optional session scope.
    /// Example: `snapshot.create_inferred_line_scoped(sketch, &input, policy, options, ids, Some(&scope))?`.
    pub fn create_inferred_line_scoped(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineCreationReport, SketchLineCreationError> {
        let (mut geometry, inference, settings) =
            self.prepare_inferred_line(sketch, input, policy, solve_options, scope)?;
        let source = if settings.auto_apply_constraints
            && (!inference.points.is_empty() || inference.direction.is_some())
        {
            Some(
                crate::sketch::SketchInferenceFreeze::capture(&geometry)
                    .map_err(SketchLineCreationError::Sketch)?,
            )
        } else {
            None
        };
        let creation = input.primitive(inference.candidate_endpoints);
        let mut report =
            append_inferred_primitive(&mut geometry, sketch, &creation, inference, &settings, ids)?;
        if let Some(source) = source {
            let relations = temporary_direction(&report, &settings)?;
            let first = geometry
                .point(report.primitive.points[0])
                .map_err(SketchLineCreationError::Sketch)?;
            let context = super::sketch_placement_recipe::inference::TransientPlacement {
                source,
                anchors: if settings.persist_inferred_constraints {
                    Vec::new()
                } else {
                    vec![*first]
                },
                relations,
            };
            if !settings.persist_inferred_constraints {
                geometry = context
                    .solve(self, &geometry, sketch, solve_options, None)
                    .map_err(|cause| inference_failure(sketch, cause))?;
            }
            report.inference_witness = Some(context);
            report.needs_preparation = settings.persist_inferred_constraints;
        }
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok(report)
    }
    fn prepare_inferred_line(
        &self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<
        (
            Sketch2D,
            SketchLinePlacementInference,
            crate::sketch_settings::SketchSettings,
        ),
        SketchLineCreationError,
    > {
        let settings = self.effective_sketch_settings();
        let geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let inference = match scope {
            Some(scope) => {
                geometry.infer_line_placement_scoped(input.endpoints, &settings, policy, scope, &[])
            }
            None => geometry.infer_line_placement(input.endpoints, &settings, policy),
        }
        .map_err(SketchLineCreationError::Sketch)?;
        Ok((geometry, inference, settings))
    }
}
impl SketchLineCreation {
    fn primitive(self, endpoints: [[f64; 2]; 2]) -> SketchPrimitiveCreation {
        SketchPrimitiveCreation::Line {
            endpoints: endpoints.map(SketchPointInput::New),
            construction: self.construction,
            centerline: self.centerline,
        }
    }
}

fn append_inferred_primitive(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    creation: &SketchPrimitiveCreation,
    inference: SketchLinePlacementInference,
    settings: &crate::sketch_settings::SketchSettings,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchLineCreationReport, SketchLineCreationError> {
    let primitive = append_primitive(geometry, sketch, creation, ids)
        .map_err(SketchLineCreationError::Primitive)?;
    let applied = relations::append(
        geometry,
        &primitive,
        &inference,
        settings.auto_apply_constraints,
        settings.persist_inferred_constraints,
        ids,
    )?;
    Ok(SketchLineCreationReport {
        primitive,
        inference,
        point_constraints: applied.0,
        direction: applied.1,
        inference_witness: None,
        needs_preparation: false,
    })
}

fn temporary_direction(
    report: &SketchLineCreationReport,
    settings: &crate::sketch_settings::SketchSettings,
) -> Result<Vec<crate::sketch::PlanarConstraint>, SketchLineCreationError> {
    if !settings.auto_apply_constraints || settings.persist_inferred_constraints {
        return Ok(Vec::new());
    }
    report
        .inference
        .direction
        .map(|direction| relations::direction_relation(report.primitive.entity, direction))
        .transpose()
        .map(|relation| relation.into_iter().collect())
}

fn inference_failure(
    sketch: DocumentSketchId,
    cause: crate::sketch::SketchTransientSolveError,
) -> SketchLineCreationError {
    match cause {
        crate::sketch::SketchTransientSolveError::NotConverged(status) => {
            SketchLineCreationError::Readiness(crate::document_sketch::SketchReadinessError {
                sketch,
                cause: crate::document_sketch::SketchReadinessCause::Unconverged(status),
            })
        }
        other => SketchLineCreationError::Inference(other),
    }
}
