// SPDX-License-Identifier: GPL-2.0-only
//! Conditional origin projection evidence bound to one borrowed document snapshot.
use super::DocumentSnapshot;
use crate::{
    document_sketch::{DocumentSketchId, SketchPlacement, SketchSolveDiagnostics},
    feature::work_geometry::WorkPointReference,
    sketch::{
        SketchEntityId, SketchPlaneRecipe, SketchSolveOutcome,
        interval_equations::enclosure::Enclosure,
    },
};

/// Caller assumptions on the current resolved frame; these are not proven document bounds.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchFrameInputBounds {
    pub origin: [[f64; 2]; 3],
    pub x_axis: [[f64; 2]; 3],
    pub y_axis: [[f64; 2]; 3],
}
/// Conditional enclosure of the model origin in one borrowed document snapshot.
pub struct ConditionalOriginProjection<'a> {
    snapshot: &'a DocumentSnapshot,
    sketch: DocumentSketchId,
    anchor: SketchEntityId,
    frame: SketchPlaneRecipe,
    assumptions: SketchFrameInputBounds,
    local: [[f64; 2]; 2],
}
impl ConditionalOriginProjection<'_> {
    /// Read the borrowed snapshot; example: `evidence.snapshot()`.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        self.snapshot
    }
    /// Read the sketch identity; example: `evidence.sketch()`.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }
    /// Read the origin source identity; example: `evidence.source()`.
    pub fn source(&self) -> WorkPointReference {
        WorkPointReference::Origin
    }
    /// Read the projected anchor identity; example: `evidence.anchor()`.
    pub fn anchor(&self) -> SketchEntityId {
        self.anchor
    }
    /// Read the resolved frame snapshot; example: `evidence.frame()`.
    pub fn frame(&self) -> SketchPlaneRecipe {
        self.frame
    }
    /// Read caller assumptions; example: `evidence.assumptions()`.
    pub fn assumptions(&self) -> SketchFrameInputBounds {
        self.assumptions
    }
    /// Read conditional local position bounds; example: `evidence.local_bounds()`.
    pub fn local_bounds(&self) -> [[f64; 2]; 2] {
        self.local
    }
}
impl DocumentSnapshot {
    /// Bound an origin binding under explicit resolved-frame assumptions and a current solve.
    /// Example: `snapshot.conditional_origin_projection(sketch, anchor, &outcome, bounds)?`.
    /// This is conditional evidence, not document or feature qualification.
    pub fn conditional_origin_projection<'a>(
        &'a self,
        sketch: DocumentSketchId,
        anchor: SketchEntityId,
        outcome: &SketchSolveOutcome,
        assumptions: SketchFrameInputBounds,
    ) -> Result<ConditionalOriginProjection<'a>, String> {
        let view = self
            .require_converged_resolved_sketch(sketch)
            .map_err(|e| e.to_string())?;
        if !matches!(view.sketch.placement(), SketchPlacement::Fixed(_))
            || view.sketch.orientation().is_some()
        {
            return Err(format!(
                "sketch {sketch:?}: expected a fixed frame without derived orientation"
            ));
        }
        if !view.sketch.solve_frame_is_current() {
            return Err(format!(
                "sketch {sketch:?}: expected the unchanged solve frame"
            ));
        }
        let binding = view
            .sketch
            .point_projections()
            .iter()
            .find(|binding| binding.anchor == anchor)
            .ok_or_else(|| {
                format!("sketch {sketch:?} anchor {anchor:?}: expected a current origin binding")
            })?;
        if binding.reference != WorkPointReference::Origin {
            return Err(format!(
                "projection {:?}: expected the model origin source",
                binding.reference
            ));
        }
        if !view.sketch.geometry().same_recipe(outcome.candidate())
            || !same_solve(view.verification, outcome)
            || !outcome.source_stamp_current(&self.parameters)
        {
            return Err(format!(
                "sketch {sketch:?}: expected the unchanged accepted solve and parameter sources"
            ));
        }
        let frame = view.plane.recipe();
        validate_frame(frame, assumptions)?;
        let current = self
            .project_datum_point_into_sketch(sketch, WorkPointReference::Origin)
            .map_err(|e| e.to_string())?;
        let local = project_bounds(assumptions, current.local_position)?;
        Ok(ConditionalOriginProjection {
            snapshot: self,
            sketch,
            anchor,
            frame,
            assumptions,
            local,
        })
    }
}
fn same_solve(diagnostics: &SketchSolveDiagnostics, outcome: &SketchSolveOutcome) -> bool {
    let numerical = outcome.numerical();
    diagnostics.termination == numerical.termination
        && diagnostics.structure == numerical.structure
        && diagnostics.conflicts == outcome.conflicting_relations()
        && diagnostics.relative_residual.to_bits() == numerical.relative_residual.to_bits()
        && diagnostics.residual_evidence == numerical.residual_evidence
        && diagnostics.residual_rows == outcome.residual_rows()
}
fn validate_frame(
    frame: SketchPlaneRecipe,
    assumptions: SketchFrameInputBounds,
) -> Result<(), String> {
    for (kind, values, boxes) in [
        ("origin", frame.origin, assumptions.origin),
        ("x axis", frame.x_axis, assumptions.x_axis),
        ("y axis", frame.y_axis, assumptions.y_axis),
    ] {
        for (value, bounds) in values.into_iter().zip(boxes) {
            Enclosure::new(bounds)?;
            if value < bounds[0] || value > bounds[1] {
                return Err(format!(
                    "frame {kind} value {value:?} and bounds {bounds:?}: expected current value inside bounds"
                ));
            }
        }
    }
    Ok(())
}
fn project_bounds(
    frame: SketchFrameInputBounds,
    current: [f64; 2],
) -> Result<[[f64; 2]; 2], String> {
    Ok([
        project_axis(frame.origin, frame.x_axis)?
            .include(current[0])?
            .bounds(),
        project_axis(frame.origin, frame.y_axis)?
            .include(current[1])?
            .bounds(),
    ])
}
fn project_axis(origin: [[f64; 2]; 3], axis: [[f64; 2]; 3]) -> Result<Enclosure, String> {
    let zero = Enclosure::new([0., 0.])?;
    origin
        .into_iter()
        .zip(axis)
        .try_fold(zero, |sum, (coordinate, direction)| {
            sum.sum(
                zero.difference(Enclosure::new(coordinate)?)?
                    .product(Enclosure::new(direction)?)?,
            )
        })
}
