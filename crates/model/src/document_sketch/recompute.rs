// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchHealth, SketchProjectionInputs,
    point_projection::UnavailableProjectionInputs,
};
use crate::{
    parameter::ParameterCollection,
    sketch::{SketchError, SketchPoint2D, SketchSolveOutcome},
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};

/// Local numerical results retain authored and structural relation diagnostics.
#[derive(Clone, Debug)]
pub struct DocumentSketchSolveReport {
    pub sketch: DocumentSketchId,
    pub result: Result<SketchSolveOutcome, SketchError>,
}
/// Ordered results; cancellation means no geometry from this batch was adopted.
#[derive(Clone, Debug)]
pub struct DocumentSketchRecompute {
    pub reports: Vec<DocumentSketchSolveReport>,
    pub cancelled: bool,
}
/// Owned candidate and its report; failure of one sketch does not stop independent sketches.
#[derive(Clone, Debug)]
pub struct DocumentSketchSolveOutcome {
    pub candidate: DocumentSketches,
    pub report: DocumentSketchRecompute,
}
impl DocumentSketches {
    /// Solve one stable identity without updating unrelated readers; example: `sketches.solve_one(id, &parameters, options, || false)?`.
    pub fn solve_one(
        &self,
        sketch: DocumentSketchId,
        parameters: &ParameterCollection<u64>,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchSolveOutcome, SketchError> {
        self.get(sketch)?;
        Ok(self.solve_scope(
            parameters,
            options,
            cancelled,
            SketchSolveScope::One(sketch),
            &UnavailableProjectionInputs,
        ))
    }
    /// Recompute independent sketches without changing the input; example: `sketches.solve(&parameters, options, || false)`.
    pub fn solve(
        &self,
        parameters: &ParameterCollection<u64>,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> DocumentSketchSolveOutcome {
        self.solve_scope(
            parameters,
            options,
            cancelled,
            SketchSolveScope::All,
            &UnavailableProjectionInputs,
        )
    }
    /// Solve only unverified or changed readers; example: `sketches.solve_pending(&parameters, options, || false)`.
    pub fn solve_pending(
        &self,
        parameters: &ParameterCollection<u64>,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> DocumentSketchSolveOutcome {
        self.solve_scope(
            parameters,
            options,
            cancelled,
            SketchSolveScope::Pending,
            &UnavailableProjectionInputs,
        )
    }
    pub(crate) fn solve_scope(
        &self,
        parameters: &ParameterCollection<u64>,
        options: SolveOptions,
        mut cancelled: impl FnMut() -> bool,
        scope: SketchSolveScope,
        inputs: &dyn SketchProjectionInputs,
    ) -> DocumentSketchSolveOutcome {
        let mut candidate = self.clone();
        let mut stopped = false;
        let mut poll = || {
            stopped |= cancelled();
            stopped
        };
        let reports = solve_ordered(
            &mut candidate,
            parameters,
            options,
            &mut poll,
            scope,
            inputs,
        );
        let cancelled = poll();
        finish_batch(self, candidate, reports, cancelled)
    }
}
fn adopt_converged(placed: &mut PlacedSketch, result: &Result<SketchSolveOutcome, SketchError>) {
    if let Ok(outcome) = result
        && outcome.numerical().termination == SolveTermination::Converged
    {
        placed.geometry = outcome.candidate().clone();
    }
}

fn solve_ordered(
    candidate: &mut DocumentSketches,
    parameters: &ParameterCollection<u64>,
    options: SolveOptions,
    poll: &mut impl FnMut() -> bool,
    scope: SketchSolveScope,
    inputs: &dyn SketchProjectionInputs,
) -> Vec<DocumentSketchSolveReport> {
    let mut reports = Vec::new();
    for placed in &mut candidate.sketches {
        if poll() {
            break;
        }
        if let Some(report) = solve_in_scope(placed, parameters, options, &mut *poll, scope, inputs)
        {
            reports.push(report);
        }
    }
    reports
}
fn solve_in_scope(
    placed: &mut PlacedSketch,
    parameters: &ParameterCollection<u64>,
    options: SolveOptions,
    poll: &mut impl FnMut() -> bool,
    scope: SketchSolveScope,
    inputs: &dyn SketchProjectionInputs,
) -> Option<DocumentSketchSolveReport> {
    if !scope.includes(placed.id) {
        return None;
    }
    let projections = placed.projection_inputs(inputs);
    if !scope.requires(placed, parameters, &projections) {
        return None;
    }
    Some(solve_placed(placed, parameters, projections, options, poll))
}

fn solve_placed(
    placed: &mut PlacedSketch,
    parameters: &ParameterCollection<u64>,
    projections: Result<Vec<SketchPoint2D>, SketchError>,
    options: SolveOptions,
    poll: &mut impl FnMut() -> bool,
) -> DocumentSketchSolveReport {
    let result = projections
        .as_ref()
        .map_err(Clone::clone)
        .and_then(|anchors| {
            placed
                .geometry
                .solve_with_grounded_points(parameters, anchors, options, poll)
        });
    adopt_converged(placed, &result);
    placed.record_evaluation(parameters, projections, &result);
    DocumentSketchSolveReport {
        sketch: placed.id,
        result,
    }
}

fn finish_batch(
    original: &DocumentSketches,
    candidate: DocumentSketches,
    reports: Vec<DocumentSketchSolveReport>,
    cancelled: bool,
) -> DocumentSketchSolveOutcome {
    // Cancellation is batch-wide: never expose partly updated document geometry (#3721).
    let candidate = if cancelled {
        original.clone()
    } else {
        candidate
    };
    DocumentSketchSolveOutcome {
        candidate,
        report: DocumentSketchRecompute { reports, cancelled },
    }
}

#[derive(Clone, Copy)]
pub(crate) enum SketchSolveScope {
    All,
    Pending,
    One(DocumentSketchId),
}
impl SketchSolveScope {
    fn includes(self, sketch: DocumentSketchId) -> bool {
        !matches!(self, Self::One(selected) if selected != sketch)
    }
    fn requires(
        self,
        placed: &PlacedSketch,
        parameters: &ParameterCollection<u64>,
        projections: &Result<Vec<SketchPoint2D>, SketchError>,
    ) -> bool {
        match self {
            Self::All => true,
            Self::One(_) => true,
            Self::Pending => !matches!(
                placed.health_with_projection_inputs(parameters, projections),
                PlacedSketchHealth::Current(_)
            ),
        }
    }
}
