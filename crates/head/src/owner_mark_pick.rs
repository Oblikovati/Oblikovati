// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, viewport face/sketch picking, and connected face chaining for Part Modify Mark (#3918).
use super::*;
use oblikovati_gui::{GuiMarkEditSeed, GuiMarkFacePick, GuiMarkGeometrySet, GuiMarkSketchPick};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, mark::MarkMethod, part::PartFeatureDefinition},
    history::DocumentSnapshot,
    sketch::{SketchEntityId, SketchPlaneRecipe},
};
use std::collections::BTreeMap;

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiMarkEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Mark(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                GuiMarkEditSeed {
                    body: definition.body().to_vec(),
                    geometry_sets: definition
                        .geometry_sets()
                        .iter()
                        .map(|set| GuiMarkGeometrySet {
                            sketch: Some(set.sketch),
                            curves: set.curves.clone(),
                            mark_type: set.mark_type,
                        })
                        .collect(),
                    method: definition.method(),
                    faces: definition.faces().to_vec(),
                    automatic_face_chain: definition.automatic_face_chain(),
                    direction: definition.direction(),
                },
            ))
        })
        .collect()
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_mark_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.mark.active {
            return Ok(None);
        }
        let (method, automatic_face_chain) = self
            .create_features
            .mark
            .preview
            .as_ref()
            .map(|p| (p.request.method, p.request.automatic_face_chain))
            .unwrap_or((MarkMethod::ProjectToClosestFace, false));
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if method == MarkMethod::WrapToFace {
            let allowed = oblikovati_session::selection::SelectionFilter::only([
                oblikovati_session::selection::SelectionKind::Face,
            ]);
            let candidates =
                self.part_scenes
                    .part_candidates(&self.files, document, ray, &allowed)?;
            if let Ok(Some(picked)) =
                candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
                && let crate::part_scene::pick_candidates::PickTarget::Face(_, key) = picked.target
            {
                let pick =
                    self.resolve_mark_face_pick(document, generation, key, automatic_face_chain)?;
                self.create_features.mark.face_pick = Some(pick);
                return Ok(Some(true));
            }
        }
        if let Some(pick) = self.pick_visible_mark_sketch_curve(document, generation, ray)? {
            self.create_features.mark.sketch_pick = Some(pick);
            return Ok(Some(true));
        }
        Ok(Some(false))
    }

    pub(super) fn resolve_mark_face_pick(
        &self,
        document: u64,
        generation: u64,
        seed_face: Vec<u8>,
        automatic_face_chain: bool,
    ) -> Result<GuiMarkFacePick, String> {
        self.require_browser_document(document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != generation {
            return Err("mark pick: expected the current document generation".into());
        }
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            if body.is_surface() {
                continue;
            }
            let Ok(keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if !keys.iter().any(|k| k.as_slice() == seed_face.as_slice()) {
                continue;
            }
            let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
            let faces = if automatic_face_chain {
                body.connected_face_chain(session.history(), guid, &seed_face)?
            } else {
                vec![seed_face.clone()]
            };
            return Ok(GuiMarkFacePick {
                document,
                generation,
                body: body_key,
                seed_face,
                faces,
                automatic_face_chain,
            });
        }
        Err("mark pick: expected a live named solid face".into())
    }

    fn pick_visible_mark_sketch_curve(
        &self,
        document: u64,
        generation: u64,
        ray: [[f64; 3]; 2],
    ) -> Result<Option<GuiMarkSketchPick>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let evaluation = self.part_scenes.current(session)?;
        let datums = evaluation.datum_evaluation();
        let mut best: Option<(f64, DocumentSketchId, SketchEntityId)> = None;
        for sketch in snapshot.sketches.sketches() {
            if !sketch.visible() {
                continue;
            }
            let Ok(resolved_plane) =
                snapshot.resolve_sketch_plane_with_datums(sketch.id(), &datums)
            else {
                continue;
            };
            let plane = resolved_plane.recipe();
            let geom = sketch.geometry();
            for line in geom.lines() {
                let (Ok(a), Ok(b)) = (geom.point(line.endpoints[0]), geom.point(line.endpoints[1]))
                else {
                    continue;
                };
                let dist = ray_segment_distance(
                    ray,
                    to_world(plane, a.position),
                    to_world(plane, b.position),
                );
                if dist < 0.35 && best.is_none_or(|(d, _, _)| dist < d) {
                    best = Some((dist, sketch.id(), line.id));
                }
            }
        }
        Ok(best.map(|(_, sketch, curve)| GuiMarkSketchPick {
            document,
            generation,
            sketch,
            curves: vec![curve],
        }))
    }
}

fn to_world(plane: SketchPlaneRecipe, pt: [f64; 2]) -> [f64; 3] {
    [
        plane.origin[0] + pt[0] * plane.x_axis[0] + pt[1] * plane.y_axis[0],
        plane.origin[1] + pt[0] * plane.x_axis[1] + pt[1] * plane.y_axis[1],
        plane.origin[2] + pt[0] * plane.x_axis[2] + pt[1] * plane.y_axis[2],
    ]
}

fn ray_segment_distance(ray: [[f64; 3]; 2], p0: [f64; 3], p1: [f64; 3]) -> f64 {
    let ro = ray[0];
    let rd = [
        ray[1][0] - ray[0][0],
        ray[1][1] - ray[0][1],
        ray[1][2] - ray[0][2],
    ];
    let rd_len = (rd[0] * rd[0] + rd[1] * rd[1] + rd[2] * rd[2]).sqrt();
    if rd_len <= 1e-12 {
        return f64::INFINITY;
    }
    let dir = [rd[0] / rd_len, rd[1] / rd_len, rd[2] / rd_len];
    let mut best = f64::INFINITY;
    for step in 0..=8 {
        let t = step as f64 / 8.0;
        let q = [
            p0[0] + (p1[0] - p0[0]) * t,
            p0[1] + (p1[1] - p0[1]) * t,
            p0[2] + (p1[2] - p0[2]) * t,
        ];
        let rel = [q[0] - ro[0], q[1] - ro[1], q[2] - ro[2]];
        let proj = (rel[0] * dir[0] + rel[1] * dir[1] + rel[2] * dir[2]).max(0.0);
        let closest = [
            ro[0] + proj * dir[0],
            ro[1] + proj * dir[1],
            ro[2] + proj * dir[2],
        ];
        let dx = q[0] - closest[0];
        let dy = q[1] - closest[1];
        let dz = q[2] - closest[2];
        best = best.min((dx * dx + dy * dy + dz * dz).sqrt());
    }
    best
}
