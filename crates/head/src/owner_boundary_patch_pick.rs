// SPDX-License-Identifier: GPL-2.0-only
//! Visible sketch curve ray picking, loop chaining, and interior diagonal rejection for Boundary Patch (#3918).
use super::super::*;
use oblikovati_automation::PartUpdateAccess;
use oblikovati_gui::{GuiBoundaryCurveRef, GuiBoundaryPatchPick, GuiBoundaryPatchSelectionMode};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    sketch::{Sketch2D, SketchCircle2D, SketchEntityId, SketchPlane},
};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_boundary_patch_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.boundary_patch.active {
            return Ok(None);
        }
        let Some((generation, edges)) =
            self.pick_visible_boundary_sketch_edges(document, ray, tolerance)?
        else {
            return Ok(Some(false));
        };
        if edges.is_empty() {
            return Ok(Some(false));
        }
        self.create_features.boundary_patch.pick_seq = self
            .create_features
            .boundary_patch
            .pick_seq
            .saturating_add(1);
        self.create_features.boundary_patch.pick = Some(GuiBoundaryPatchPick {
            document,
            generation,
            sequence: self.create_features.boundary_patch.pick_seq,
            mode: self.create_features.boundary_patch.selection_mode,
            edges,
        });
        Ok(Some(true))
    }

    fn pick_visible_boundary_sketch_edges(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<(u64, Vec<GuiBoundaryCurveRef>)>, String> {
        let (generation, visible_sketches): (u64, Vec<(DocumentSketchId, SketchPlacement)>) = {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            let list = session
                .snapshot()
                .sketches
                .sketches()
                .iter()
                .filter(|placed| placed.visible())
                .map(|placed| (placed.id(), placed.placement()))
                .collect();
            (session.generation(), list)
        };
        let base_datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let hit_tol = tolerance.unwrap_or(0.08).max(0.12);
        let mut best: Option<(f64, DocumentSketchId, SketchEntityId)> = None;
        for (sketch_id, placement) in visible_sketches {
            let mut local_datums = base_datums.clone();
            let needs_face_lookup = {
                let session = self
                    .files
                    .workspace()
                    .session(document)
                    .map_err(|cause| cause.to_string())?;
                session
                    .snapshot()
                    .require_converged_resolved_sketch_with_datums(sketch_id, &local_datums)
                    .is_err()
            };
            if needs_face_lookup
                && let SketchPlacement::Face(ref face_ref) = placement
                && let Ok(current) = self.part_scenes.current_part_datums_for_face(
                    &mut self.files,
                    document,
                    face_ref,
                )
                && let Ok(session) = self.files.workspace().session(document)
                && let Ok(datums) = current.current(session)
            {
                local_datums = datums.clone();
            }
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            let snapshot = session.snapshot();
            let plane = match snapshot
                .require_converged_resolved_sketch_with_datums(sketch_id, &local_datums)
            {
                Ok(resolved) => resolved.plane,
                Err(_) => match snapshot.resolve_sketch_plane(sketch_id) {
                    Ok(plane) => plane,
                    Err(_) => continue,
                },
            };
            let Ok(placed) = snapshot.sketches.get(sketch_id) else {
                continue;
            };
            if let Some((dist, entity)) =
                nearest_sketch_curve_hit(plane, placed.geometry(), ray, hit_tol)
                && best.is_none_or(|(best_d, _, _)| dist < best_d)
            {
                best = Some((dist, sketch_id, entity));
            }
        }
        let Some((_, sketch_id, entity)) = best else {
            return Ok(None);
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let Ok(placed) = session.snapshot().sketches.get(sketch_id) else {
            return Ok(None);
        };
        let geometry = placed.geometry();
        if oblikovati_martensite::is_interior_diagonal_curve(geometry, entity) {
            return Ok(Some((generation, Vec::new())));
        }
        let mode = self.create_features.boundary_patch.selection_mode;
        let auto_chain = self.create_features.boundary_patch.automatic_edge_chain;
        let entities = if mode == GuiBoundaryPatchSelectionMode::Boundary && auto_chain {
            oblikovati_martensite::chain_sketch_boundary_loop(geometry, entity)
        } else {
            vec![entity]
        };
        let refs = entities
            .into_iter()
            .map(|id| GuiBoundaryCurveRef::SketchCurve {
                sketch: sketch_id,
                entity: id,
            })
            .collect();
        Ok(Some((generation, refs)))
    }
}

fn nearest_sketch_curve_hit(
    plane: SketchPlane,
    geometry: &Sketch2D,
    ray: [[f64; 3]; 2],
    tolerance: f64,
) -> Option<(f64, SketchEntityId)> {
    let mut best: Option<(f64, SketchEntityId)> = None;
    for line in geometry.lines() {
        if !matches!(geometry.is_construction(line.id), Ok(false)) {
            continue;
        }
        let (Ok(p0), Ok(p1)) = (
            geometry.point(line.endpoints[0]),
            geometry.point(line.endpoints[1]),
        ) else {
            continue;
        };
        let (Ok(start), Ok(end)) = (plane.to_model(p0.position), plane.to_model(p1.position))
        else {
            continue;
        };
        if let Some(dist) = ray_segment_distance(ray, [start, end])
            && dist <= tolerance
            && best.is_none_or(|(best_d, _)| dist < best_d)
        {
            best = Some((dist, line.id));
        }
    }
    for circle in geometry.circles() {
        if !matches!(geometry.is_construction(circle.id), Ok(false)) {
            continue;
        }
        if let Some(dist) = nearest_circle_hit(geometry, plane, circle, ray)
            && dist <= tolerance
            && best.is_none_or(|(best_d, _)| dist < best_d)
        {
            best = Some((dist, circle.id));
        }
    }
    best
}

fn nearest_circle_hit(
    geometry: &Sketch2D,
    plane: SketchPlane,
    circle: &SketchCircle2D,
    ray: [[f64; 3]; 2],
) -> Option<f64> {
    let center = geometry.point(circle.center).ok()?.position;
    let steps = 24usize;
    let mut best: Option<f64> = None;
    for i in 0..steps {
        let a0 = (i as f64 / steps as f64) * std::f64::consts::TAU;
        let a1 = ((i + 1) as f64 / steps as f64) * std::f64::consts::TAU;
        let p0 = plane
            .to_model([
                center[0] + circle.radius * a0.cos(),
                center[1] + circle.radius * a0.sin(),
            ])
            .ok()?;
        let p1 = plane
            .to_model([
                center[0] + circle.radius * a1.cos(),
                center[1] + circle.radius * a1.sin(),
            ])
            .ok()?;
        if let Some(dist) = ray_segment_distance(ray, [p0, p1])
            && best.is_none_or(|b| dist < b)
        {
            best = Some(dist);
        }
    }
    best
}

fn ray_segment_distance(
    [origin, direction]: [[f64; 3]; 2],
    [start, end]: [[f64; 3]; 2],
) -> Option<f64> {
    let dot = |a: [f64; 3], b: [f64; 3]| (0..3).map(|i| a[i] * b[i]).sum::<f64>();
    let dir_sq = dot(direction, direction);
    if dir_sq <= 0.0 || !dir_sq.is_finite() {
        return None;
    }
    let perp = |point: [f64; 3]| {
        let offset = std::array::from_fn(|i| point[i] - origin[i]);
        let t = dot(offset, direction) / dir_sq;
        std::array::from_fn(|i| offset[i] - t * direction[i])
    };
    let a = perp(start);
    let b = perp(end);
    let seg = std::array::from_fn(|i| b[i] - a[i]);
    let seg_sq = dot(seg, seg);
    let frac = if seg_sq > 0.0 {
        (-dot(a, seg) / seg_sq).clamp(0.0, 1.0)
    } else {
        0.0
    };
    let nearest = std::array::from_fn(|i| a[i] + frac * seg[i]);
    Some(dot(nearest, nearest).sqrt())
}
