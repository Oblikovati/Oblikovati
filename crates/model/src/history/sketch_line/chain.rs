// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    history::{SketchDimensionCreation, SketchDimensionCreationReport, SketchPlacementFieldLock},
    sketch::{
        DistanceMeasure, MeasuredDimensionOperands, SeededDimensionOperands,
        SketchDimensionalConstraintCreation, SketchEntityId, SketchInferenceCandidates,
        chain_layout::{line_chain_indices, require_chain_vertices},
    },
    sketch_settings::SketchSettings,
};

/// Continuous Line tool vertices; a closed two-point chain remains one open segment (Go #2024).
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineChainCreation {
    pub points: Vec<[f64; 2]>,
    pub closed: bool,
    pub construction: bool,
    pub centerline: bool,
    /// Existing vertices the first and last points join, so a chain stays connected when
    /// inference is off and each segment keeps its own undo step.
    pub joins: [Option<SketchEntityId>; 2],
    /// Typed length and angle of the last segment; each becomes a driving dimension.
    pub locks: [Option<SketchPlacementFieldLock>; 2],
}
/// Visible vertices and ordered line reports preserve shared identity and per-segment inference.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineChainCreationReport {
    pub sketch: DocumentSketchId,
    pub points: Vec<SketchEntityId>,
    pub lines: Vec<SketchLineCreationReport>,
    pub dimensions: Vec<SketchDimensionCreationReport>,
}
impl DocumentSnapshot {
    /// Author the continuous Line tool's visible chain atomically against current solved geometry.
    /// Example: `snapshot.create_inferred_sketch_line_chain(sketch, &input, policy, options, ids)?;`.
    pub fn create_inferred_sketch_line_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        self.create_inferred_sketch_line_chain_scoped(
            sketch,
            input,
            policy,
            solve_options,
            ids,
            None,
        )
    }
    /// Author a chain with an explicit inference scope; example: `snapshot.create_inferred_sketch_line_chain_scoped(sketch, &input, policy, options, ids, Some(&scope))?`.
    pub fn create_inferred_sketch_line_chain_scoped(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        let needs_choice = input.locks.iter().any(Option::is_some)
            && self.effective_sketch_settings().over_constrained_behavior
                == oblikovati_api::types::OVER_CONSTRAINED_PROMPT;
        if !needs_choice {
            return self.append_inferred_chain(sketch, input, policy, solve_options, ids, scope);
        }
        let mut candidate = self.clone();
        let result =
            candidate.append_inferred_chain(sketch, input, policy, solve_options, ids, scope)?;
        self.sketches = candidate.sketches;
        self.parameters = candidate.parameters;
        Ok(result)
    }
    fn append_inferred_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        require_chain_vertices(&input.points, false).map_err(SketchLineCreationError::Sketch)?;
        policy
            .require_valid()
            .map_err(SketchLineCreationError::Sketch)?;
        let mut geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let report = input.append_chain(
            &mut geometry,
            sketch,
            &self.effective_sketch_settings(),
            policy,
            ids,
            scope,
        )?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        let dimensions =
            self.append_locked_dimensions(sketch, input, &report, solve_options, ids)?;
        Ok(SketchLineChainCreationReport {
            dimensions,
            ..report
        })
    }
    fn append_locked_dimensions(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        report: &SketchLineChainCreationReport,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<Vec<SketchDimensionCreationReport>, SketchLineCreationError> {
        let Some(last) = report.lines.last() else {
            return Ok(Vec::new());
        };
        let operands = locked_segment_operands(last);
        let mut dimensions = Vec::new();
        for (index, (lock, operands)) in input.locks.iter().zip(operands).enumerate() {
            let Some(lock) = lock else {
                continue;
            };
            let creation = SketchDimensionCreation {
                operands,
                expression: lock.expression.clone(),
                reference: lock.reference,
                text_point: lock.text_point,
            };
            let created = self
                .create_placement_dimension(sketch, &creation, options, ids, None)
                .map_err(|cause| SketchLineCreationError::Dimension { index, cause })?;
            dimensions.push(created);
        }
        Ok(dimensions)
    }
}
fn locked_segment_operands(
    line: &SketchLineCreationReport,
) -> [SketchDimensionalConstraintCreation; 2] {
    let [start, end] = [line.primitive.points[0], line.primitive.points[1]];
    [
        SketchDimensionalConstraintCreation::Measured(MeasuredDimensionOperands::Distance {
            points: [start, end],
            measure: DistanceMeasure::Aligned,
        }),
        SketchDimensionalConstraintCreation::Seeded(SeededDimensionOperands::LineAngle(
            line.primitive.entity,
        )),
    ]
}
struct LineInferenceContext<'a> {
    settings: &'a SketchSettings,
    policy: SketchInferencePolicy,
    scope: Option<&'a SketchInferenceCandidates>,
    current: &'a [SketchEntityId],
}
impl SketchLineChainCreation {
    fn append_chain(
        &self,
        geometry: &mut Sketch2D,
        sketch: DocumentSketchId,
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        ids: &mut RecipeReservations<'_>,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        let first = match self.joins[0] {
            Some(vertex) => vertex,
            None => append_visible_vertex(geometry, sketch, self.points[0], ids)?,
        };
        let mut report = SketchLineChainCreationReport {
            sketch,
            points: vec![first],
            lines: Vec::new(),
            dimensions: Vec::new(),
        };
        let closed = self.closed && self.points.len() >= 3;
        for indices in line_chain_indices(self.points.len(), closed) {
            if indices[1] == report.points.len() {
                let vertex = self.chain_vertex(geometry, sketch, indices[1], ids)?;
                report.points.push(vertex);
            }
            let endpoints = indices.map(|index| report.points[index]);
            let current: Vec<_> = report
                .points
                .iter()
                .copied()
                .chain(report.lines.iter().map(|line| line.primitive.entity))
                .collect();
            report.lines.push(self.append_segment(
                geometry,
                sketch,
                endpoints,
                ids,
                LineInferenceContext {
                    settings,
                    policy,
                    scope,
                    current: &current,
                },
            )?);
        }
        Ok(report)
    }
    fn chain_vertex(
        &self,
        geometry: &mut Sketch2D,
        sketch: DocumentSketchId,
        index: usize,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchEntityId, SketchLineCreationError> {
        if index + 1 == self.points.len()
            && let Some(vertex) = self.joins[1]
        {
            return Ok(vertex);
        }
        append_visible_vertex(geometry, sketch, self.points[index], ids)
    }
    fn append_segment(
        &self,
        geometry: &mut Sketch2D,
        sketch: DocumentSketchId,
        points: [SketchEntityId; 2],
        ids: &mut RecipeReservations<'_>,
        context: LineInferenceContext<'_>,
    ) -> Result<SketchLineCreationReport, SketchLineCreationError> {
        let LineInferenceContext {
            settings,
            policy,
            scope,
            current,
        } = context;
        let inference = match scope {
            Some(scope) => {
                geometry.infer_owned_line_placement_scoped(points, settings, policy, scope, current)
            }
            None => geometry.infer_owned_line_placement(points, settings, policy),
        };
        let mut inference = inference.map_err(SketchLineCreationError::Sketch)?;
        // A typed angle owns the direction; an inferred relation on top would be redundant.
        if self.locks[1].is_some() {
            inference.direction = None;
        }
        let creation = SketchPrimitiveCreation::Line {
            endpoints: points.map(SketchPointInput::Owned),
            construction: self.construction,
            centerline: self.centerline,
        };
        append_inferred_primitive(
            geometry,
            sketch,
            &creation,
            inference,
            settings.auto_apply_constraints,
            ids,
        )
    }
}
fn append_visible_vertex(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    position: [f64; 2],
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchEntityId, SketchLineCreationError> {
    let primitive = SketchPrimitiveCreation::Point { position };
    append_primitive(geometry, sketch, &primitive, ids)
        .map(|report| report.entity)
        .map_err(SketchLineCreationError::Primitive)
}
