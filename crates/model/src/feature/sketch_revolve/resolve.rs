// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    feature::{FeatureReadFootprint, FeatureReadKey},
    history::{DocumentSnapshot, ResolvedDocumentSketch},
};

/// Current input borrowed from the document; the axis is re-read from its sole centerline.
#[derive(Debug)]
pub struct ResolvedSketchRevolve<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub selection: &'a SketchPrismSelection,
    pub axis: [[f64; 2]; 2],
    pub angle: f64,
}

impl SketchRevolveDefinition {
    /// Resolve current fully constrained profile, sole centerline axis, and angle.
    /// Example: `definition.resolve(&snapshot)?` before native construction.
    pub fn resolve<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
    ) -> Result<ResolvedSketchRevolve<'a>, SketchRevolveError> {
        let sketch = snapshot
            .require_resolved_sketch(self.sketch)
            .map_err(|cause| self.decline(SketchRevolveCause::Placement(cause)))?;
        let axis = current_axis(sketch.sketch.geometry()).map_err(|cause| self.decline(cause))?;
        let angle = self
            .angle
            .resolve(&snapshot.parameters)
            .map_err(|cause| self.decline(SketchRevolveCause::Angle(cause)))?;
        Ok(ResolvedSketchRevolve {
            sketch,
            selection: &self.selection,
            axis,
            angle,
        })
    }

    /// Attribute sketch and angle dependencies; example: `definition.reads(&snapshot)?`.
    pub fn reads(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<FeatureReadFootprint, SketchRevolveError> {
        let sketch = snapshot
            .sketches
            .get(self.sketch)
            .map_err(|cause| self.decline(SketchRevolveCause::Sketch(cause)))?;
        let mut keys = sketch.read_footprint().keys().clone();
        keys.extend(self.angle.parameter_id().map(FeatureReadKey::Parameter));
        Ok(FeatureReadFootprint::new(
            keys,
            sketch.read_footprint().has_unattributed_reads(),
        ))
    }
}

fn current_axis(geometry: &crate::sketch::Sketch2D) -> Result<[[f64; 2]; 2], SketchRevolveCause> {
    let line = geometry
        .centerline_axis()
        .map_err(SketchRevolveCause::Sketch)?
        .ok_or_else(|| {
            SketchRevolveCause::Definition(
                "revolve centerline: expected exactly one owned centerline".into(),
            )
        })?;
    let first = geometry
        .point(line.endpoints[0])
        .map_err(SketchRevolveCause::Sketch)?
        .position;
    let second = geometry
        .point(line.endpoints[1])
        .map_err(SketchRevolveCause::Sketch)?
        .position;
    Ok([first, second])
}
