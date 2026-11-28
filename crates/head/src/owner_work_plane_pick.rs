// SPDX-License-Identifier: GPL-2.0-only
//! Viewport geometry picking (vertices, edges, faces, work planes) for Work Plane creation (#3918).

use super::*;
use oblikovati_gui::{GuiWorkPlaneMode, GuiWorkPlanePick, GuiWorkPlaneViewportPick};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_work_plane_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(mode) = self.create_features.work_plane.active_mode else {
            return Ok(None);
        };
        let tolerance = depth_window.unwrap_or(0.5).max(0.35);
        let mut picks = Vec::new();
        match mode {
            GuiWorkPlaneMode::ThreePoints => {
                self.append_vertex_picks(document, ray, tolerance, &mut picks)?;
            }
            GuiWorkPlaneMode::TwoCoplanarEdges => {
                self.append_edge_picks(document, ray, tolerance, &mut picks)?;
            }
            GuiWorkPlaneMode::MidplaneTwoPlanes => {
                self.append_plane_or_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::ParallelThroughPoint => {
                self.append_vertex_picks(document, ray, tolerance, &mut picks)?;
                self.append_plane_or_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::NormalToAxisThroughPoint | GuiWorkPlaneMode::LineThroughPoint => {
                self.append_vertex_picks(document, ray, tolerance, &mut picks)?;
                self.append_edge_picks(document, ray, tolerance, &mut picks)?;
            }
            GuiWorkPlaneMode::AngleToPlaneAroundEdge => {
                self.append_edge_picks(document, ray, tolerance, &mut picks)?;
                self.append_plane_or_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::MidplaneOfTorus => {
                self.append_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::TangentThroughEdge => {
                self.append_edge_picks(document, ray, tolerance, &mut picks)?;
                self.append_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::TangentThroughPoint => {
                self.append_vertex_picks(document, ray, tolerance, &mut picks)?;
                self.append_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::TangentParallelToPlane => {
                self.append_plane_or_face_picks(document, ray, &mut picks)?;
            }
            GuiWorkPlaneMode::General => {
                self.append_vertex_picks(document, ray, tolerance, &mut picks)?;
                self.append_edge_picks(document, ray, tolerance, &mut picks)?;
                self.append_plane_or_face_picks(document, ray, &mut picks)?;
            }
        }
        if picks.is_empty() {
            return Ok(Some(false));
        }
        self.create_features.work_plane.pick_seq += 1;
        let sequence = self.create_features.work_plane.pick_seq;
        self.create_features.work_plane.last_pick = Some(GuiWorkPlaneViewportPick {
            sequence,
            document,
            candidates: picks,
        });
        Ok(Some(true))
    }

    fn append_vertex_picks(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: f64,
        out: &mut Vec<GuiWorkPlanePick>,
    ) -> Result<(), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let mut hits = Vec::new();
        for (_, body) in self.part_scenes.current(session)?.ordered_body_outputs() {
            for hit in body.named_vertex_hits_on_ray(session.history(), guid, ray, tolerance)? {
                hits.push((hit.distance, hit.reach, hit.key));
            }
        }
        hits.sort_by(|a, b| a.0.total_cmp(&b.0).then_with(|| a.1.total_cmp(&b.1)));
        for (_, _, key) in hits {
            out.push(GuiWorkPlanePick::TopologyVertex(key));
        }
        Ok(())
    }

    fn append_edge_picks(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: f64,
        out: &mut Vec<GuiWorkPlanePick>,
    ) -> Result<(), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let mut hits = Vec::new();
        for (_, body) in self.part_scenes.current(session)?.ordered_body_outputs() {
            for hit in body.edge_hits_on_ray(session.history(), guid, ray, tolerance)? {
                if hit.distance <= tolerance {
                    hits.push((hit.distance, hit.reach, hit.key));
                }
            }
        }
        hits.sort_by(|a, b| a.0.total_cmp(&b.0).then_with(|| a.1.total_cmp(&b.1)));
        for (_, _, key) in hits {
            out.push(GuiWorkPlanePick::TopologyEdge(key));
        }
        Ok(())
    }

    fn append_face_picks(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        out: &mut Vec<GuiWorkPlanePick>,
    ) -> Result<(), String> {
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        for target in candidates.into_sorted_targets() {
            if let crate::part_scene::pick_candidates::PickTarget::Face(_, key) = target {
                out.push(GuiWorkPlanePick::TopologyFace(key));
            }
        }
        Ok(())
    }

    fn append_plane_or_face_picks(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        out: &mut Vec<GuiWorkPlanePick>,
    ) -> Result<(), String> {
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::WorkPlane,
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let mut candidates =
            self.part_scenes
                .part_candidates(&self.files, document, ray, &allowed)?;
        self.append_visible_plane_hits(document, ray, &mut candidates)?;
        for target in candidates.into_sorted_targets() {
            match target {
                crate::part_scene::pick_candidates::PickTarget::WorkPlane(id) => {
                    out.push(GuiWorkPlanePick::WorkPlane(id));
                }
                crate::part_scene::pick_candidates::PickTarget::Face(_, key) => {
                    out.push(GuiWorkPlanePick::TopologyFace(key));
                }
                _ => {}
            }
        }
        Ok(())
    }
}
