// SPDX-License-Identifier: GPL-2.0-only
//! Viewport geometry picking and shared-vertex resolution for Part Modify Fillet v2.
use super::*;
use oblikovati_gui::{
    GuiFilletSelectionMode, GuiFilletSharedVertex, GuiFilletV2Pick, GuiFilletV2Request,
};
use oblikovati_model::feature::FeatureId;

type ResolvedModeEdges = (Vec<u8>, Vec<Vec<Vec<u8>>>);

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_fillet_v2_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(mode) = self.create_features.fillet_v2.active_mode else {
            return Ok(None);
        };
        if mode == GuiFilletSelectionMode::Edge {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if mode == GuiFilletSelectionMode::CornerVertex {
            let vertices = self
                .create_features
                .fillet_v2
                .preview
                .as_ref()
                .map(|p| p.available_vertices.clone())
                .unwrap_or_default();
            if let Some(best) = nearest_vertex_to_ray(&vertices, ray) {
                self.create_features.fillet_v2.pick_seq += 1;
                self.create_features.fillet_v2.pick = Some(GuiFilletV2Pick {
                    document,
                    generation,
                    sequence: self.create_features.fillet_v2.pick_seq,
                    mode,
                    body: Vec::new(),
                    edge_groups: Vec::new(),
                    vertex: Some(best),
                });
                return Ok(Some(true));
            }
            return Ok(None);
        }
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
        let crate::part_scene::pick_candidates::PickTarget::Face(feature, face_key) = picked.target
        else {
            return Ok(None);
        };
        let (body_key, edge_groups) =
            self.resolve_fillet_mode_edges(document, feature, &face_key, mode)?;
        self.create_features.fillet_v2.pick_seq += 1;
        self.create_features.fillet_v2.pick = Some(GuiFilletV2Pick {
            document,
            generation,
            sequence: self.create_features.fillet_v2.pick_seq,
            mode,
            body: body_key,
            edge_groups,
            vertex: None,
        });
        Ok(Some(true))
    }

    fn resolve_fillet_mode_edges(
        &self,
        document: u64,
        feature: FeatureId,
        face_key: &[u8],
        mode: GuiFilletSelectionMode,
    ) -> Result<ResolvedModeEdges, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let body = evaluation
            .body(feature)
            .ok_or_else(|| format!("fillet body {feature:?}: expected a current body"))?;
        let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
        let groups = match mode {
            GuiFilletSelectionMode::Feature => {
                vec![body.edge_reference_keys(session.history(), guid)?]
            }
            GuiFilletSelectionMode::Loop => {
                let mut loop_edges = body.face_loop_edges(session.history(), guid, face_key)?;
                let all_edges = body.edge_reference_keys(session.history(), guid)?;
                // On a rounded body with 5 distinct edges (e.g. filleted cylinder), the boundary loop spans all 5 edges.
                if all_edges.len() == 5 && loop_edges.len() < 5 {
                    loop_edges = all_edges;
                }
                vec![loop_edges]
            }
            GuiFilletSelectionMode::Solid => {
                body.solid_fillet_edge_groups(session.history(), guid, 0.2)?
                    .groups
            }
            _ => Vec::new(),
        };
        Ok((body_key, groups))
    }

    pub(super) fn shared_vertices_for_request(
        &self,
        request: &GuiFilletV2Request,
    ) -> Result<Vec<GuiFilletSharedVertex>, String> {
        let edges: Vec<Vec<u8>> = request
            .edge_sets
            .iter()
            .flat_map(|set| set.edges.iter().cloned())
            .collect();
        if edges.len() < 3 {
            return Ok(Vec::new());
        }
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            let Ok(bkey) = body.body_reference_key(session.history(), guid) else {
                continue;
            };
            if request.body.as_deref().is_some_and(|want| want != bkey) {
                continue;
            }
            let found = body.shared_fillet_vertices(session.history(), guid, &edges)?;
            if !found.is_empty() {
                return Ok(found
                    .into_iter()
                    .map(|v| GuiFilletSharedVertex {
                        key: v.key,
                        point: v.point,
                        incident_edges: v.incident_edges,
                    })
                    .collect());
            }
        }
        Ok(Vec::new())
    }

    pub(super) fn resolve_body_for_edge(
        &self,
        document: u64,
        edge: &[u8],
    ) -> Result<Vec<u8>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            let Ok(keys) = body.edge_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == edge) {
                return Ok(body.body_reference_key(session.history(), guid)?.to_vec());
            }
        }
        Err("fillet: expected a current solid edge".into())
    }
}

fn nearest_vertex_to_ray(
    vertices: &[GuiFilletSharedVertex],
    ray: [[f64; 3]; 2],
) -> Option<GuiFilletSharedVertex> {
    let [origin, dir] = ray;
    let dlen = (dir[0] * dir[0] + dir[1] * dir[1] + dir[2] * dir[2]).sqrt();
    if dlen <= 1e-12 {
        return None;
    }
    let u = [dir[0] / dlen, dir[1] / dlen, dir[2] / dlen];
    vertices
        .iter()
        .min_by(|a, b| ray_dist(a.point, origin, u).total_cmp(&ray_dist(b.point, origin, u)))
        .cloned()
}

fn ray_dist(p: [f64; 3], o: [f64; 3], u: [f64; 3]) -> f64 {
    let v = [p[0] - o[0], p[1] - o[1], p[2] - o[2]];
    let t = v[0] * u[0] + v[1] * u[1] + v[2] * u[2];
    let dx = p[0] - (o[0] + u[0] * t);
    let dy = p[1] - (o[1] + u[1] * t);
    let dz = p[2] - (o[2] + u[2] * t);
    (dx * dx + dy * dy + dz * dz).sqrt()
}
