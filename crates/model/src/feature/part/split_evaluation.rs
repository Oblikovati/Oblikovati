// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Modify Split features (face splits and multi-body solid splits, #3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_lineage::BodySlot,
    body_resolution::{resolve_body_face, resolve_body_slot},
};
use crate::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    feature::{
        FeatureId,
        hole::HoleOriginPlane,
        split::{
            ResolvedSplit, ResolvedSplitCurve, ResolvedSplitTarget, ResolvedSplitTool,
            SplitDefinition, SplitKeepSide, SplitMode, SplitTarget, SplitTool,
        },
        work_plane::WorkPlaneResolver,
    },
    history::DocumentSnapshot,
    sketch::{SketchEntityId, SketchPlane},
};
use std::collections::BTreeSet;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_split<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &SplitDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let tool = self
            .resolve_split_tool(snapshot, backend, definition.tool())
            .map_err(PartFeatureFailure::SplitInput)?;
        let target = match definition.target() {
            SplitTarget::AllFaces { .. } => ResolvedSplitTarget::AllFaces,
            SplitTarget::Faces { faces, .. } => ResolvedSplitTarget::Faces(faces.clone()),
            SplitTarget::Solid { .. } => ResolvedSplitTarget::Solid,
        };
        let input = ResolvedSplit {
            mode: definition.mode(),
            tool,
            target,
            keep_side: definition.keep_side(),
        };
        let body = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "split body slot {slot:?}: expected a current body"
            ))
        })?;
        let mut results = backend.evaluate_split(feature, body, input)?;
        let expected_count = match (definition.mode(), definition.keep_side()) {
            (SplitMode::Solid, SplitKeepSide::Both) => 2,
            _ => 1,
        };
        if results.len() != expected_count {
            return Err(PartFeatureFailure::SplitInput(format!(
                "split outputs {}: expected {expected_count} bodies",
                results.len()
            )));
        }
        if expected_count == 1 {
            let result = results.remove(0);
            backend
                .verify_body_reference(&result, definition.body())
                .map_err(PartFeatureFailure::BodyReference)?;
            self.body_lineage
                .replace(slot, feature)
                .map_err(PartFeatureFailure::BodyReference)?;
            self.bodies.remove(&slot);
            self.bodies.insert(feature.into(), result);
            return Ok(());
        }
        let slots: Vec<_> = (0..results.len())
            .map(|index| BodySlot {
                feature,
                output: index as u32,
            })
            .collect();
        let slot_set: BTreeSet<_> = slots.iter().copied().collect();
        self.body_lineage
            .replace_with_outputs(slot, feature, slot_set)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.extend(slots.into_iter().zip(results));
        Ok(())
    }

    fn resolve_split_tool<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &B,
        tool: &SplitTool,
    ) -> Result<ResolvedSplitTool, String> {
        match tool {
            SplitTool::OriginPlane(origin) => Ok(ResolvedSplitTool::Plane(origin_plane(*origin))),
            SplitTool::WorkPlane(plane_id) => self
                .datums
                .plane(*plane_id)
                .map(ResolvedSplitTool::Plane)
                .map_err(|cause| format!("split work plane {plane_id:?}: {cause}")),
            SplitTool::SketchCurve { sketch, entity } => self
                .resolve_sketch_curve(snapshot, backend, *sketch, *entity)
                .map(ResolvedSplitTool::Curve),
        }
    }

    fn resolve_sketch_curve<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &B,
        sketch_id: DocumentSketchId,
        entity: SketchEntityId,
    ) -> Result<ResolvedSplitCurve, String> {
        let placed = snapshot
            .sketches
            .get(sketch_id)
            .map_err(|cause| format!("split sketch {sketch_id:?}: {cause}"))?;
        if let SketchPlacement::Face(face_ref) = placed.placement() {
            let plane = resolve_body_face(&self.bodies, &self.body_lineage, backend, &face_ref)
                .map_err(|cause| cause.to_string())?;
            self.datums.insert_sketch_face(&face_ref, plane);
        }
        let resolved = snapshot
            .require_converged_sketch_with_datums(sketch_id, &self.datums)
            .map_err(|cause| format!("split sketch {sketch_id:?}: {cause}"))?;
        let geometry = resolved.sketch.geometry();
        let plane = resolved.plane;
        if let Ok(line) = geometry.line(entity) {
            let p0 = geometry
                .point(line.endpoints[0])
                .map_err(|cause| cause.to_string())?;
            let p1 = geometry
                .point(line.endpoints[1])
                .map_err(|cause| cause.to_string())?;
            let start = plane.to_model(p0.position).map_err(|c| c.to_string())?;
            let end = plane.to_model(p1.position).map_err(|c| c.to_string())?;
            return Ok(ResolvedSplitCurve::Line {
                start,
                end,
                plane_normal: plane.normal(),
            });
        }
        if let Ok(arc) = geometry.arc(entity) {
            let c = geometry
                .point(arc.center)
                .map_err(|cause| cause.to_string())?;
            let s = geometry
                .point(arc.start)
                .map_err(|cause| cause.to_string())?;
            let e = geometry.point(arc.end).map_err(|cause| cause.to_string())?;
            return Ok(ResolvedSplitCurve::Arc {
                center: plane.to_model(c.position).map_err(|e| e.to_string())?,
                start: plane.to_model(s.position).map_err(|e| e.to_string())?,
                end: plane.to_model(e.position).map_err(|e| e.to_string())?,
                counter_clockwise: arc.counter_clockwise,
                plane,
            });
        }
        if let Ok(circle) = geometry.circle(entity) {
            let c = geometry
                .point(circle.center)
                .map_err(|cause| cause.to_string())?;
            return Ok(ResolvedSplitCurve::Circle {
                center: plane.to_model(c.position).map_err(|e| e.to_string())?,
                radius: circle.radius,
                plane,
            });
        }
        Err(format!(
            "split sketch {sketch_id:?}, entity {entity:?}: expected a line, arc, or circle"
        ))
    }
}

fn origin_plane(plane: HoleOriginPlane) -> SketchPlane {
    match plane {
        HoleOriginPlane::Xy => SketchPlane::xy(),
        HoleOriginPlane::Xz => SketchPlane::xz(),
        HoleOriginPlane::Yz => SketchPlane::yz(),
    }
}
