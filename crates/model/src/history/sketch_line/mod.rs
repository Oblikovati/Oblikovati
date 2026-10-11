// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationReport,
    sketch_primitive::append_primitive,
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        Sketch2D, SketchConstraintId, SketchDirectionInference, SketchInferencePolicy,
        SketchLinePlacementInference, SketchPointInput, SketchPrimitiveCreation,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;
mod error;
mod relations;
pub use error::SketchLineCreationError;
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
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineCreationReport {
    pub primitive: SketchPrimitiveCreationReport,
    pub inference: SketchLinePlacementInference,
    pub point_constraints: Vec<SketchConstraintId>,
    pub direction: Option<SketchAppliedLineDirection>,
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
        let (mut geometry, inference, enabled) =
            self.prepare_inferred_line(sketch, input, policy, solve_options)?;
        let creation = input.primitive(inference.candidate_endpoints);
        let primitive = append_primitive(&mut geometry, sketch, &creation, ids)
            .map_err(SketchLineCreationError::Primitive)?;
        let applied = relations::append(&mut geometry, &primitive, &inference, enabled, ids)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok(creation_report(primitive, inference, applied))
    }
    fn prepare_inferred_line(
        &self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
    ) -> Result<(Sketch2D, SketchLinePlacementInference, bool), SketchLineCreationError> {
        let settings = self.effective_sketch_settings();
        let geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let inference = geometry
            .infer_line_placement(input.endpoints, &settings, policy)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok((geometry, inference, settings.auto_apply_constraints))
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

fn creation_report(
    primitive: SketchPrimitiveCreationReport,
    inference: SketchLinePlacementInference,
    applied: (Vec<SketchConstraintId>, Option<SketchAppliedLineDirection>),
) -> SketchLineCreationReport {
    SketchLineCreationReport {
        primitive,
        inference,
        point_constraints: applied.0,
        direction: applied.1,
    }
}
