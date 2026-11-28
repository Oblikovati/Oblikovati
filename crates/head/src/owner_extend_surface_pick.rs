// SPDX-License-Identifier: GPL-2.0-only
//! Surface boundary edge picking, Edge Chain highlighting, and To-termination picking for Extend Surface (#3918).
use super::super::*;
use crate::owner::browser_selection::ChosenModelEdge;
use oblikovati_gui::GuiExtendSurfacePick;
use oblikovati_model::feature::{FeatureId, extend_surface::ExtendSurfaceTermination};

struct EdgeRayCandidate {
    distance: f64,
    reach: f64,
    feature: FeatureId,
    body_key: Vec<u8>,
    edge_key: Vec<u8>,
}

struct ResolvedEdgePick {
    generation: u64,
    feature: FeatureId,
    target_body: Vec<u8>,
    edges: Vec<Vec<u8>>,
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
    pub(in crate::owner) fn pick_extend_surface_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.extend_surface.active {
            return Ok(None);
        }
        if self.create_features.extend_surface.picking_termination {
            return self.pick_extend_termination(document, ray, tolerance);
        }
        let Some(resolved) = self.resolve_surface_edge_pick(document, ray, tolerance)? else {
            return Ok(None);
        };
        self.highlight_extend_edges(
            document,
            resolved.generation,
            resolved.feature,
            &resolved.edges,
        );
        self.create_features.extend_surface.pick_seq = self
            .create_features
            .extend_surface
            .pick_seq
            .saturating_add(1);
        self.create_features.extend_surface.pick = Some(GuiExtendSurfacePick {
            document,
            generation: resolved.generation,
            sequence: self.create_features.extend_surface.pick_seq,
            target_body: Some(resolved.target_body),
            edges: resolved.edges,
            termination: None,
        });
        Ok(Some(true))
    }

    fn pick_extend_termination(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
            oblikovati_session::selection::SelectionKind::WorkPlane,
        ]);
        let mut candidates =
            self.part_scenes
                .part_candidates(&self.files, document, ray, &allowed)?;
        self.append_visible_plane_hits(document, ray, &mut candidates)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let termination = match picked.target {
            crate::part_scene::pick_candidates::PickTarget::Face(feature, face_key) => {
                let eval = self.part_scenes.current(session)?;
                let Some(body) = eval.body(feature) else {
                    return Ok(None);
                };
                let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
                ExtendSurfaceTermination::Face {
                    body: body_key,
                    face: face_key,
                }
            }
            crate::part_scene::pick_candidates::PickTarget::WorkPlane(feature) => {
                ExtendSurfaceTermination::WorkPlane(feature)
            }
            _ => return Ok(None),
        };
        self.create_features.extend_surface.pick_seq = self
            .create_features
            .extend_surface
            .pick_seq
            .saturating_add(1);
        self.create_features.extend_surface.pick = Some(GuiExtendSurfacePick {
            document,
            generation,
            sequence: self.create_features.extend_surface.pick_seq,
            target_body: None,
            edges: Vec::new(),
            termination: Some(termination),
        });
        Ok(Some(true))
    }

    fn resolve_surface_edge_pick(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<ResolvedEdgePick>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let eval = self.part_scenes.current(session)?;
        let pick_tol = tolerance.unwrap_or(0.25).max(0.25);
        let mut best: Option<EdgeRayCandidate> = None;
        for (feature, body) in eval.ordered_body_outputs() {
            if !body.is_surface() {
                continue;
            }
            let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
            for hit in body.edge_hits_on_ray(session.history(), guid, ray, pick_tol)? {
                if hit.distance > pick_tol {
                    continue;
                }
                let better = best.as_ref().is_none_or(|current| {
                    hit.distance < current.distance - 1.0e-5
                        || ((hit.distance - current.distance).abs() <= 1.0e-5
                            && hit.reach < current.reach)
                });
                if better {
                    best = Some(EdgeRayCandidate {
                        distance: hit.distance,
                        reach: hit.reach,
                        feature,
                        body_key: body_key.clone(),
                        edge_key: hit.key,
                    });
                }
            }
        }
        let Some(EdgeRayCandidate {
            feature,
            body_key: target_body,
            edge_key,
            ..
        }) = best
        else {
            return Ok(None);
        };
        let Some(body) = eval.body(feature) else {
            return Ok(None);
        };
        let chain = self.create_features.extend_surface.edge_chain;
        let edges = body.surface_boundary_edge_chain(session.history(), guid, &edge_key, chain)?;
        Ok(Some(ResolvedEdgePick {
            generation,
            feature,
            target_body,
            edges,
        }))
    }

    fn highlight_extend_edges(
        &mut self,
        document: u64,
        generation: u64,
        feature: FeatureId,
        edges: &[Vec<u8>],
    ) {
        self.model_edges.clear();
        for key in edges {
            self.model_edges.push(ChosenModelEdge {
                document,
                generation,
                feature,
                key: key.clone(),
            });
        }
        self.selected_edge_outlines = None;
    }
}
