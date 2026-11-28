// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    feature::{FeatureReadFootprint, FeatureReadKey, work_geometry::WorkGeometryEvaluation},
    history::{DocumentSnapshot, ResolvedDocumentSketch},
};

/// Current input borrowed from the document; the axis is re-read from its explicit line or sole centerline.
#[derive(Debug)]
pub struct ResolvedSketchRevolve<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub selection: &'a SketchPrismSelection,
    pub axis: [[f64; 2]; 2],
    pub angle: f64,
    pub second_angle: Option<f64>,
    pub full: bool,
    pub direction: SketchRevolveDirection,
    pub surface: bool,
}

impl SketchRevolveDefinition {
    /// Resolve current fully constrained profile, axis, and angle.
    /// Example: `definition.resolve(&snapshot)?` before native construction.
    pub fn resolve<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
    ) -> Result<ResolvedSketchRevolve<'a>, SketchRevolveError> {
        let sketch = snapshot
            .require_resolved_sketch(self.sketch)
            .map_err(|cause| self.decline(SketchRevolveCause::Placement(cause)))?;
        self.resolve_on_sketch(snapshot, sketch)
    }

    /// Resolve current profile, axis, and angle using evaluated Part datums for work-plane sketches.
    /// Example: `definition.resolve_with_datums(&snapshot, &datums)?` before creation.
    pub fn resolve_with_datums<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchRevolve<'a>, SketchRevolveError> {
        let sketch = snapshot
            .require_converged_resolved_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| self.decline(SketchRevolveCause::Placement(cause)))?;
        self.resolve_on_sketch(snapshot, sketch)
    }

    pub(in crate::feature) fn resolve_for_replay<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchRevolve<'a>, SketchRevolveError> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| self.decline(SketchRevolveCause::Placement(cause)))?;
        self.resolve_on_sketch(snapshot, sketch)
    }

    fn resolve_on_sketch<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        sketch: ResolvedDocumentSketch<'a>,
    ) -> Result<ResolvedSketchRevolve<'a>, SketchRevolveError> {
        let axis = current_axis(sketch.sketch.geometry(), self.axis)
            .map_err(|cause| self.decline(cause))?;
        let (angle, second_angle) = if self.full {
            (std::f64::consts::TAU, None)
        } else {
            let primary = self
                .angle
                .resolve(&snapshot.parameters)
                .map_err(|cause| self.decline(SketchRevolveCause::Angle(cause)))?;
            let secondary = self
                .second_angle
                .map(|value| {
                    value
                        .resolve(&snapshot.parameters)
                        .map_err(|cause| self.decline(SketchRevolveCause::Angle(cause)))
                })
                .transpose()?;
            (primary, secondary)
        };
        Ok(ResolvedSketchRevolve {
            sketch,
            selection: &self.selection,
            axis,
            angle,
            second_angle,
            full: self.full,
            direction: self.direction,
            surface: self.surface,
        })
    }

    /// Attribute sketch, Boolean target, and angle dependencies; example: `definition.reads(&snapshot)?`.
    pub fn reads(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<FeatureReadFootprint, SketchRevolveError> {
        let sketch = snapshot
            .sketches
            .get(self.sketch)
            .map_err(|cause| self.decline(SketchRevolveCause::Sketch(cause)))?;
        let mut keys = sketch.read_footprint().keys().clone();
        keys.insert(FeatureReadKey::Sketch(self.sketch.0));
        keys.extend(self.angle.parameter_id().map(FeatureReadKey::Parameter));
        keys.extend(
            self.second_angle
                .and_then(DatumAngle::parameter_id)
                .map(FeatureReadKey::Parameter),
        );
        if let Some((_, _, source)) = self.boolean_target() {
            keys.insert(FeatureReadKey::Feature(source));
        }
        Ok(FeatureReadFootprint::new(
            keys,
            sketch.read_footprint().has_unattributed_reads(),
        ))
    }
}

fn current_axis(
    geometry: &crate::sketch::Sketch2D,
    explicit: Option<SketchEntityId>,
) -> Result<[[f64; 2]; 2], SketchRevolveCause> {
    let endpoints = match explicit {
        Some(entity) => {
            geometry
                .line(entity)
                .map_err(SketchRevolveCause::Sketch)?
                .endpoints
        }
        None => {
            geometry
                .centerline_axis()
                .map_err(SketchRevolveCause::Sketch)?
                .ok_or_else(|| {
                    SketchRevolveCause::Definition(
                        "revolve centerline: expected exactly one owned centerline".into(),
                    )
                })?
                .endpoints
        }
    };
    let first = geometry
        .point(endpoints[0])
        .map_err(SketchRevolveCause::Sketch)?
        .position;
    let second = geometry
        .point(endpoints[1])
        .map_err(SketchRevolveCause::Sketch)?
        .position;
    Ok([first, second])
}
