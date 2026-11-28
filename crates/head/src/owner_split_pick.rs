// SPDX-License-Identifier: GPL-2.0-only
//! Viewport geometry picking (sketch curves, work planes, faces, solids) for Split (#3918).
use super::seed;
use crate::owner::*;
use oblikovati_automation::PartUpdateAccess;
use oblikovati_gui::{GuiSplitPick, GuiSplitPickKind, GuiSplitPickMode};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    feature::split::{SplitMode, SplitTool},
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
    pub(in crate::owner) fn pick_split_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.split.active {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let kind = match (
            self.create_features.split.pick_mode,
            self.create_features.split.split_mode,
        ) {
            (GuiSplitPickMode::Tool, _) => self.pick_split_tool_on_ray(document, ray, tolerance)?,
            (GuiSplitPickMode::Target, SplitMode::Faces) => {
                self.pick_split_face_on_ray(document, ray, tolerance)?
            }
            (GuiSplitPickMode::Target, SplitMode::Solid) => {
                self.pick_split_solid_on_ray(document, ray, tolerance)?
            }
        };
        let Some(kind) = kind else {
            return Ok(None);
        };
        self.create_features.split.pick_seq = self.create_features.split.pick_seq.saturating_add(1);
        self.create_features.split.pick = Some(GuiSplitPick {
            document,
            generation,
            sequence: self.create_features.split.pick_seq,
            kind,
        });
        Ok(Some(true))
    }

    fn pick_split_tool_on_ray(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSplitPickKind>, String> {
        if let Some(curve) = self.pick_visible_sketch_curve(document, ray, tolerance)? {
            return Ok(Some(curve));
        }
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::WorkPlane,
        ]);
        let mut candidates = crate::part_scene::pick_candidates::PickCandidates::new(
            crate::part_scene::pick_candidates::PickCoverage::VisiblePlanesAndNearestFaces,
        );
        self.append_visible_plane_hits(document, ray, &mut candidates)?;
        if let Ok(Some(hit)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
            && let crate::part_scene::pick_candidates::PickTarget::WorkPlane(feature) = hit.target
        {
            return Ok(Some(GuiSplitPickKind::WorkPlane(feature)));
        }
        Ok(None)
    }

    fn pick_visible_sketch_curve(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSplitPickKind>, String> {
        let visible_sketches: Vec<(DocumentSketchId, SketchPlacement)> = {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            session
                .snapshot()
                .sketches
                .sketches()
                .iter()
                .filter(|placed| placed.visible())
                .map(|placed| (placed.id(), placed.placement()))
                .collect()
        };
        let base_datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let hit_tol = tolerance.unwrap_or(0.08).max(0.12);
        let mut best: Option<(f64, SplitTool)> = None;
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
            let Ok(resolved) = session
                .snapshot()
                .require_converged_resolved_sketch_with_datums(sketch_id, &local_datums)
            else {
                continue;
            };
            let geometry = resolved.sketch.geometry();
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
                let (Ok(start), Ok(end)) = (
                    resolved.plane.to_model(p0.position),
                    resolved.plane.to_model(p1.position),
                ) else {
                    continue;
                };
                if let Some(frac) =
                    super::super::owner_sweep::ray_segment_fraction(ray, [start, end], hit_tol)
                {
                    let pt = [
                        start[0] + frac * (end[0] - start[0]),
                        start[1] + frac * (end[1] - start[1]),
                        start[2] + frac * (end[2] - start[2]),
                    ];
                    let depth = (pt[0] - ray[0][0]) * ray[1][0]
                        + (pt[1] - ray[0][1]) * ray[1][1]
                        + (pt[2] - ray[0][2]) * ray[1][2];
                    if best.as_ref().is_none_or(|(d, _)| depth < *d) {
                        best = Some((
                            depth,
                            SplitTool::SketchCurve {
                                sketch: sketch_id,
                                entity: line.id,
                            },
                        ));
                    }
                }
            }
        }
        let Some((_, tool)) = best else {
            return Ok(None);
        };
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let bodies: Vec<_> = evaluation.ordered_bodies().collect();
        let (default_body, default_face_count) = if bodies.len() == 1 {
            let body_key = bodies[0]
                .body_reference_key(session.history(), guid)
                .ok()
                .map(|k| k.to_vec());
            let faces = seed::pre_split_face_count(bodies[0], session.history(), guid);
            (body_key, faces)
        } else {
            (None, 0)
        };
        Ok(Some(GuiSplitPickKind::SketchCurve {
            tool,
            default_body,
            default_face_count,
        }))
    }

    fn pick_split_face_on_ray(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSplitPickKind>, String> {
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(_, key) = picked.target else {
            return Ok(None);
        };
        let (body, face) = self.resolve_split_body_and_face(document, &key)?;
        Ok(Some(GuiSplitPickKind::Face { body, face }))
    }

    fn pick_split_solid_on_ray(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSplitPickKind>, String> {
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Body,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Body(_, hex_key) = picked.target else {
            return Ok(None);
        };
        let body = seed::decode_hex_key(&hex_key)?;
        Ok(Some(GuiSplitPickKind::Solid { body }))
    }

    fn resolve_split_body_and_face(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<(Vec<u8>, Vec<u8>), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for (feature, solid) in evaluation.ordered_body_outputs() {
            let Ok(keys) = solid.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == face) {
                let body_key = solid.body_reference_key(session.history(), guid)?.to_vec();
                let resolved_face = if self.create_features.split.editing == Some(feature) {
                    seed::remap_split_descendant_face(solid, face)
                } else {
                    face.to_vec()
                };
                return Ok((body_key, resolved_face));
            }
        }
        Err("split: expected a current solid face".into())
    }
}
