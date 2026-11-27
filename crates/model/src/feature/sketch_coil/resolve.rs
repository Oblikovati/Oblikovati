// SPDX-License-Identifier: GPL-2.0-only
//! Resolve authored Coil inputs against current sketch geometry, datums, and parameters.
use super::*;
use crate::{
    feature::{FeatureReadFootprint, work_geometry::WorkGeometryEvaluation},
    history::{DocumentSnapshot, ResolvedDocumentSketch},
};

const EPSILON: f64 = 1e-9;

/// Current evaluated Coil inputs ready for certified Martensite construction.
#[derive(Debug)]
pub struct ResolvedSketchCoil<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub selection: &'a SketchPrismSelection,
    pub axis_origin: [f64; 3],
    pub axis_direction: [f64; 3],
    pub pitch: f64,
    pub revolutions: f64,
    pub taper: f64,
    pub spiral: bool,
    pub handedness: CoilHandedness,
    pub start_end: Option<(f64, f64)>,
    pub end_end: Option<(f64, f64)>,
}

impl SketchCoilDefinition {
    /// Resolve current profile, axis, and sizing parameters using evaluated Part datums.
    pub fn resolve_with_datums<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchCoil<'a>, String> {
        let sketch = snapshot
            .require_converged_resolved_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| format!("coil sketch: {cause}"))?;
        self.resolve_on_sketch(snapshot, datums, sketch)
    }

    pub(in crate::feature) fn resolve_for_replay<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchCoil<'a>, String> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| format!("coil sketch: {cause}"))?;
        self.resolve_on_sketch(snapshot, datums, sketch)
    }

    fn resolve_on_sketch<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
        sketch: ResolvedDocumentSketch<'a>,
    ) -> Result<ResolvedSketchCoil<'a>, String> {
        let (axis_origin, mut axis_direction) = resolve_axis(self.axis, &sketch, datums)?;
        if self.axis_reversed {
            axis_direction = axis_direction.map(|v| -v);
        }
        let resolve_len = |value: Option<DatumLength>, name: &str| -> Result<f64, String> {
            let len = value
                .ok_or_else(|| format!("coil {name}: expected a positive length"))?
                .resolve(&snapshot.parameters)
                .map_err(|cause| format!("coil {name}: {cause}"))?;
            if !len.is_finite() || len <= EPSILON {
                return Err(format!("coil {name} {len}: expected a positive length"));
            }
            Ok(len)
        };
        let resolve_revs = || -> Result<f64, String> {
            let revs = self
                .revolutions
                .ok_or_else(|| "coil revolutions: expected a positive turn count".to_owned())?;
            if !revs.is_finite() || revs <= EPSILON {
                return Err(format!(
                    "coil revolutions {revs}: expected a positive turn count"
                ));
            }
            Ok(revs)
        };
        let (pitch, revolutions, spiral) = match self.method {
            CoilMethod::PitchAndRevolution => {
                (resolve_len(self.pitch, "pitch")?, resolve_revs()?, false)
            }
            CoilMethod::RevolutionAndHeight => {
                let revs = resolve_revs()?;
                let height = resolve_len(self.height, "height")?;
                (height / revs, revs, false)
            }
            CoilMethod::PitchAndHeight => {
                let pitch = resolve_len(self.pitch, "pitch")?;
                let height = resolve_len(self.height, "height")?;
                (pitch, height / pitch, false)
            }
            CoilMethod::Spiral => (resolve_len(self.pitch, "pitch")?, resolve_revs()?, true),
        };
        let taper = self
            .taper
            .resolve(&snapshot.parameters)
            .map_err(|cause| format!("coil taper: {cause}"))?;
        if !taper.is_finite() || taper.abs() >= std::f64::consts::FRAC_PI_2 - 1e-4 {
            return Err(format!(
                "coil taper {taper}: expected an angle strictly within (-90 deg, 90 deg)"
            ));
        }
        if spiral && taper.abs() > EPSILON {
            return Err("coil: spiral method does not support taper".into());
        }
        let resolve_end =
            |end: Option<CoilEndTreatment>, label: &str| -> Result<Option<(f64, f64)>, String> {
                let Some(end) = end else {
                    return Ok(None);
                };
                let flat = end
                    .flat_angle
                    .resolve(&snapshot.parameters)
                    .map_err(|cause| format!("coil {label} flat angle: {cause}"))?;
                let transition = end
                    .transition_angle
                    .resolve(&snapshot.parameters)
                    .map_err(|cause| format!("coil {label} transition angle: {cause}"))?;
                if !flat.is_finite() || flat < 0.0 || !transition.is_finite() || transition < 0.0 {
                    return Err(format!(
                        "coil {label} end angles: expected non-negative finite angles"
                    ));
                }
                Ok(Some((flat, transition)))
            };
        Ok(ResolvedSketchCoil {
            sketch,
            selection: &self.selection,
            axis_origin,
            axis_direction,
            pitch,
            revolutions,
            taper: if spiral { 0.0 } else { taper },
            spiral,
            handedness: self.handedness,
            start_end: resolve_end(self.start_end, "start")?,
            end_end: resolve_end(self.end_end, "end")?,
        })
    }

    /// Attribute sketch, axis, Boolean target, and parameter dependencies.
    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let sketch = snapshot
            .sketches
            .get(self.sketch)
            .map_err(|cause| format!("coil sketch: {cause}"))?;
        let mut keys = sketch.read_footprint().keys().clone();
        keys.extend(self.source_reads().keys().iter().copied());
        Ok(FeatureReadFootprint::new(
            keys,
            sketch.read_footprint().has_unattributed_reads(),
        ))
    }
}

fn resolve_axis(
    axis: CoilAxisSource,
    sketch: &ResolvedDocumentSketch<'_>,
    datums: &WorkGeometryEvaluation,
) -> Result<([f64; 3], [f64; 3]), String> {
    match axis {
        CoilAxisSource::WorkAxis(reference) => {
            let line = reference
                .line(datums)
                .map_err(|cause| format!("coil axis: {cause}"))?;
            Ok((line.origin(), line.direction()))
        }
        CoilAxisSource::SketchLine(entity) => {
            let geometry = sketch.sketch.geometry();
            let line = geometry
                .line(entity)
                .map_err(|cause| format!("coil axis line: {cause}"))?;
            let p0 = geometry
                .point(line.endpoints[0])
                .map_err(|cause| format!("coil axis start: {cause}"))?
                .position;
            let p1 = geometry
                .point(line.endpoints[1])
                .map_err(|cause| format!("coil axis end: {cause}"))?
                .position;
            let start = sketch
                .plane
                .to_model(p0)
                .map_err(|cause| format!("coil axis start: {cause}"))?;
            let end = sketch
                .plane
                .to_model(p1)
                .map_err(|cause| format!("coil axis end: {cause}"))?;
            let delta = [end[0] - start[0], end[1] - start[1], end[2] - start[2]];
            let len = (delta[0] * delta[0] + delta[1] * delta[1] + delta[2] * delta[2]).sqrt();
            if !len.is_finite() || len <= EPSILON {
                return Err("coil axis line: expected a nonzero length line".into());
            }
            Ok((start, [delta[0] / len, delta[1] / len, delta[2] / len]))
        }
    }
}
