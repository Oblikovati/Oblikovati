// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    feature::{
        FeatureReadFootprint, FeatureReadKey, work_geometry::WorkGeometryEvaluation,
        work_plane::WorkPlaneResolver,
    },
    history::{DocumentSnapshot, ResolvedDocumentSketch},
    sketch::SketchPlane,
};
/// Current immutable inputs borrowed from their owners; never a saved frame or a stored kernel result.
#[derive(Debug)]
pub struct ResolvedSketchPrism<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub sweep_start: SketchPlane,
    pub selection: &'a SketchPrismSelection,
    pub displacement: [f64; 3],
}
impl SketchPrismDefinition {
    /// Require current converged geometry, placement and finite length inputs without side effects.
    /// Example: `let input = definition.resolve(&snapshot)?; input.sketch.plane`.
    pub fn resolve<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
    ) -> Result<ResolvedSketchPrism<'a>, SketchPrismError> {
        let sketch = snapshot
            .require_converged_resolved_sketch(self.sketch)
            .map_err(|cause| self.decline(SketchPrismCause::Placement(cause)))?;
        self.resolve_current_sketch(
            snapshot,
            sketch,
            &WorkGeometryEvaluation::default(),
            extent::missing_target_bounds,
        )
    }
    pub(crate) fn resolve_with_datums<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
        resolver: &impl WorkPlaneResolver,
        target_span: impl FnOnce(SketchPlane, ThroughAllDirection) -> Result<(f64, f64), String>,
    ) -> Result<ResolvedSketchPrism<'a>, SketchPrismError> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| self.decline(SketchPrismCause::Placement(cause)))?;
        self.resolve_current_sketch(snapshot, sketch, resolver, target_span)
    }
    fn resolve_current_sketch<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        sketch: ResolvedDocumentSketch<'a>,
        resolver: &impl WorkPlaneResolver,
        target_span: impl FnOnce(SketchPlane, ThroughAllDirection) -> Result<(f64, f64), String>,
    ) -> Result<ResolvedSketchPrism<'a>, SketchPrismError> {
        let mut displacement = self.resolve_displacement(snapshot)?;
        let (offset, depth) = if let Some(extent) = &self.extent {
            extent
                .span(sketch.plane, resolver, target_span)
                .map_err(|cause| self.decline(cause))?
        } else {
            self.validate_normal_source(displacement[2])
                .map_err(|cause| self.decline(cause))?;
            self.normal_span(snapshot, displacement[2])?
        };
        displacement[2] = depth;
        let sweep_start = self.sweep_start(sketch.plane, offset)?;
        Ok(ResolvedSketchPrism {
            sketch,
            sweep_start,
            selection: &self.selection,
            displacement,
        })
    }
    fn normal_span(
        &self,
        snapshot: &DocumentSnapshot,
        depth: f64,
    ) -> Result<(f64, f64), SketchPrismError> {
        if let Some(source) = self.second_depth {
            let second = source.resolve(&snapshot.parameters).map_err(|cause| {
                self.decline(SketchPrismCause::Length {
                    axis: 3,
                    source,
                    cause,
                })
            })?;
            validation::require_second_depth(second).map_err(|cause| self.decline(cause))?;
            let span = depth + second;
            if !span.is_finite() {
                return Err(self.decline(SketchPrismCause::Definition(format!(
                    "prism depths {depth:?} and {second:?}: expected a finite total span"
                ))));
            }
            return Ok((-second, span));
        }
        Ok(match self.direction {
            SketchPrismDirection::Negative => (0., -depth),
            SketchPrismDirection::Symmetric => (-depth / 2., depth),
            _ => (0., depth),
        })
    }
    fn sweep_start(
        &self,
        plane: SketchPlane,
        offset: f64,
    ) -> Result<SketchPlane, SketchPrismError> {
        if offset == 0. {
            return Ok(plane);
        }
        let mut start = plane.recipe();
        let normal = plane.normal();
        start.origin = std::array::from_fn(|axis| normal[axis].mul_add(offset, start.origin[axis]));
        SketchPlane::from_recipe(start, plane.options())
            .map_err(|cause| self.decline(SketchPrismCause::Sketch(cause)))
    }
    fn resolve_displacement(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<[f64; 3], SketchPrismError> {
        let mut values = [0.; 3];
        for (axis, source) in self.displacement.into_iter().enumerate() {
            values[axis] = source.resolve(&snapshot.parameters).map_err(|cause| {
                self.decline(SketchPrismCause::Length {
                    axis,
                    source,
                    cause,
                })
            })?;
        }
        Ok(values)
    }
    /// Attribute the sketch, placement, dimensions and displacement sources in the current document.
    /// Example: `definition.reads(&snapshot)?` for the shared feature graph.
    pub fn reads(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<FeatureReadFootprint, SketchPrismError> {
        let sketch = snapshot
            .sketches
            .get(self.sketch)
            .map_err(|cause| self.decline(SketchPrismCause::Sketch(cause)))?;
        let reads = sketch.read_footprint();
        let mut keys = reads.keys().clone();
        keys.extend(self.source_reads().keys());
        Ok(FeatureReadFootprint::new(
            keys,
            reads.has_unattributed_reads(),
        ))
    }
}

impl SketchPrismDefinition {
    /// Attribute direct authored sources without requiring evaluated geometry; example: `definition.source_reads()` before program restore.
    pub fn source_reads(&self) -> FeatureReadFootprint {
        let keys = std::iter::once(FeatureReadKey::Sketch(self.sketch.0))
            .chain(
                self.displacement
                    .into_iter()
                    .chain(self.second_depth)
                    .filter_map(DatumLength::parameter)
                    .map(FeatureReadKey::Parameter),
            )
            .chain(self.extent.iter().flat_map(SketchPrismExtent::reads))
            .chain(
                self.boolean_target()
                    .map(|target| FeatureReadKey::Feature(target.source())),
            );
        FeatureReadFootprint::new(keys, false)
    }
}
