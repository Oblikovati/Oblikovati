// SPDX-License-Identifier: GPL-2.0-only
//! Analytic viewport picking for vertices, edges, planar faces, and revolved faces (#3918).
use super::*;
use oblikovati_gui::GuiWorkAxisMode;
use oblikovati_martensite::AnalyticEdgeAxisKind;
use oblikovati_model::{feature::work_axis::WorkAxisLine, sketch::SketchPlane};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_work_axis_geometry(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(active) = self.create_features.work_axis.active.clone() else {
            return Ok(None);
        };
        if active.document != document {
            return Ok(None);
        }
        let base_tol = depth_window.unwrap_or(0.12).max(0.05);
        let vertex_tol = (base_tol * 2.5).max(0.15);
        let edge_tol = (base_tol * 1.5).max(0.08);
        let changed = match active.mode {
            GuiWorkAxisMode::ThroughTwoPoints => {
                if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
                    self.feed_work_axis_point(document, point)?
                } else {
                    self.keep_work_axis_prompt()?
                }
            }
            GuiWorkAxisMode::OnLineOrEdge => {
                if let Some(axis) = self.nearest_edge_axis_on_ray(
                    document,
                    line,
                    edge_tol,
                    Some(AnalyticEdgeAxisKind::Straight),
                )? {
                    self.commit_work_axis_line(document, axis)?
                } else {
                    self.keep_work_axis_prompt()?
                }
            }
            GuiWorkAxisMode::ThroughCircularEdge => {
                if let Some(axis) = self.nearest_edge_axis_on_ray(
                    document,
                    line,
                    edge_tol,
                    Some(AnalyticEdgeAxisKind::CircularOrElliptical),
                )? {
                    self.commit_work_axis_line(document, axis)?
                } else {
                    self.keep_work_axis_prompt()?
                }
            }
            GuiWorkAxisMode::IntersectionOfTwoPlanes => {
                if let Some(plane) = self.nearest_plane_on_ray(document, line)? {
                    self.feed_work_axis_plane(document, plane)?
                } else {
                    self.keep_work_axis_prompt()?
                }
            }
            GuiWorkAxisMode::NormalToPlaneThroughPoint => {
                self.pick_normal_plane_or_point(document, line, vertex_tol, &active)?
            }
            GuiWorkAxisMode::ParallelToLineThroughPoint => {
                self.pick_parallel_line_or_point(document, line, vertex_tol, edge_tol, &active)?
            }
            GuiWorkAxisMode::ThroughRevolvedFace => {
                if let Some(axis) = self.nearest_revolved_face_axis(document, line)? {
                    self.commit_work_axis_line(document, axis)?
                } else {
                    self.keep_work_axis_prompt()?
                }
            }
            GuiWorkAxisMode::General => {
                self.pick_general_work_axis(document, line, vertex_tol, edge_tol)?
            }
        };
        Ok(Some(changed))
    }

    fn pick_normal_plane_or_point(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        vertex_tol: f64,
        active: &ActiveWorkAxis,
    ) -> Result<bool, String> {
        if active.first_plane.is_some() {
            if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
                return self.feed_work_axis_point(document, point);
            }
            return self.keep_work_axis_prompt();
        }
        if active.first_point.is_some() {
            if let Some(plane) = self.nearest_plane_on_ray(document, line)? {
                return self.feed_work_axis_plane(document, plane);
            }
            return self.keep_work_axis_prompt();
        }
        if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
            return self.feed_work_axis_point(document, point);
        }
        if let Some(plane) = self.nearest_plane_on_ray(document, line)? {
            return self.feed_work_axis_plane(document, plane);
        }
        self.keep_work_axis_prompt()
    }

    fn pick_parallel_line_or_point(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        vertex_tol: f64,
        edge_tol: f64,
        active: &ActiveWorkAxis,
    ) -> Result<bool, String> {
        if active.first_line.is_some() {
            if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
                return self.feed_work_axis_point(document, point);
            }
            return self.keep_work_axis_prompt();
        }
        if active.first_point.is_some() {
            if let Some(axis) = self.nearest_edge_axis_on_ray(
                document,
                line,
                edge_tol,
                Some(AnalyticEdgeAxisKind::Straight),
            )? {
                return self.feed_work_axis_line(document, axis);
            }
            return self.keep_work_axis_prompt();
        }
        if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
            return self.feed_work_axis_point(document, point);
        }
        if let Some(axis) = self.nearest_edge_axis_on_ray(
            document,
            line,
            edge_tol,
            Some(AnalyticEdgeAxisKind::Straight),
        )? {
            return self.feed_work_axis_line(document, axis);
        }
        self.keep_work_axis_prompt()
    }

    fn pick_general_work_axis(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        vertex_tol: f64,
        edge_tol: f64,
    ) -> Result<bool, String> {
        if let Some(point) = self.nearest_vertex_on_ray(document, line, vertex_tol)? {
            return self.feed_work_axis_point(document, point);
        }
        if let Some(axis) = self.nearest_edge_axis_on_ray(document, line, edge_tol, None)? {
            return self.commit_work_axis_line(document, axis);
        }
        if let Some(axis) = self.nearest_revolved_face_axis(document, line)? {
            return self.commit_work_axis_line(document, axis);
        }
        if let Some(plane) = self.nearest_plane_on_ray(document, line)? {
            return self.feed_work_axis_plane(document, plane);
        }
        self.keep_work_axis_prompt()
    }

    fn nearest_vertex_on_ray(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Option<[f64; 3]>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut best: Option<(f64, f64, [f64; 3])> = None;
        for (_, body) in evaluation.ordered_body_outputs() {
            let key = body.body_reference_key(session.history(), guid)?;
            let key_text: String = key.iter().map(|b| format!("{b:02x}")).collect();
            if !session.body_visible(&key_text) {
                continue;
            }
            for hit in body.vertex_hits_on_ray(session.history(), guid, line, tolerance)? {
                let better = best.is_none_or(|(dist, reach, _)| {
                    hit.distance < dist || (hit.distance == dist && hit.reach < reach)
                });
                if better {
                    best = Some((hit.distance, hit.reach, hit.position));
                }
            }
        }
        Ok(best.map(|(_, _, pos)| pos))
    }

    fn nearest_edge_axis_on_ray(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
        tolerance: f64,
        required_kind: Option<AnalyticEdgeAxisKind>,
    ) -> Result<Option<WorkAxisLine>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut best: Option<(f64, f64, WorkAxisLine)> = None;
        for (_, body) in evaluation.ordered_body_outputs() {
            let key = body.body_reference_key(session.history(), guid)?;
            let key_text: String = key.iter().map(|b| format!("{b:02x}")).collect();
            if !session.body_visible(&key_text) {
                continue;
            }
            for hit in body.edge_hits_on_ray(session.history(), guid, line, tolerance)? {
                if hit.distance > tolerance {
                    continue;
                }
                let Some(edge_axis) = body.edge_axis(session.history(), guid, &hit.key)? else {
                    continue;
                };
                if required_kind.is_some_and(|kind| edge_axis.kind != kind) {
                    continue;
                }
                let better = best.is_none_or(|(dist, reach, _)| {
                    hit.distance < dist || (hit.distance == dist && hit.reach < reach)
                });
                if better {
                    best = Some((hit.distance, hit.reach, edge_axis.line));
                }
            }
        }
        Ok(best.map(|(_, _, axis)| axis))
    }

    fn nearest_revolved_face_axis(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<Option<WorkAxisLine>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut hits: Vec<(f64, Option<WorkAxisLine>)> = Vec::new();
        for (_, body) in evaluation.ordered_body_outputs() {
            let key = body.body_reference_key(session.history(), guid)?;
            let key_text: String = key.iter().map(|b| format!("{b:02x}")).collect();
            if !session.body_visible(&key_text) {
                continue;
            }
            for (face_key, reach) in body.face_hits_on_line(session.history(), guid, line)? {
                let axis = body.revolved_face_axis(session.history(), guid, &face_key)?;
                hits.push((reach, axis));
            }
        }
        hits.sort_by(|a, b| a.0.total_cmp(&b.0));
        Ok(hits.into_iter().next().and_then(|(_, axis)| axis))
    }

    fn nearest_plane_on_ray(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<Option<SketchPlane>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut best: Option<(f64, SketchPlane)> = None;
        for (_, body) in evaluation.ordered_body_outputs() {
            let key = body.body_reference_key(session.history(), guid)?;
            let key_text: String = key.iter().map(|b| format!("{b:02x}")).collect();
            if !session.body_visible(&key_text) {
                continue;
            }
            for (face_key, reach) in body.face_hits_on_line(session.history(), guid, line)? {
                if let Ok(plane) = body.face_plane(session.history(), guid, &face_key)
                    && best.is_none_or(|(dist, _)| reach < dist)
                {
                    best = Some((reach, plane));
                }
            }
        }
        Ok(best.map(|(_, plane)| plane))
    }
}
